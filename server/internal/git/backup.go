package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git-backup-web/server/internal/db"
)

// Config 备份执行配置
type Config struct {
	Platform      string // github | gitcode | gitee（影响 repo URL 拼接）
	GitUser       string // 用户名 / 命名空间
	GitToken      string
	RepoName      string
	Branch        string
	BackupDir     string
	ServerName    string
	BackupSources []string
	HostRoot      string
	// RemoteURL 仅用于测试或自托管 Git 服务覆盖；为空时按 Platform 自动生成。
	RemoteURL string
}

// Logger 日志回调：level 取值 INFO/SUCCESS/WARNING/ERROR/GIT
type Logger func(level, msg string)

// ProgressFunc 上报任务当前阶段、总体百分比和用户可读说明。
type ProgressFunc func(phase string, percent int, message string)

// PlatformHost 不同平台的 git host 与 owner 段拼接规则
type platformHost struct {
	Host string // 远端主机名（不带协议）
}

// PlatformHosts 各平台的远端 host 配置
var PlatformHosts = map[string]platformHost{
	db.PlatformGitHub:  {Host: "github.com"},
	db.PlatformGitCode: {Host: "gitcode.com"},
	db.PlatformGitee:   {Host: "gitee.com"},
}

// BuildRepoURL 按平台拼接不含凭据的 HTTPS 地址。
// Token 不写入 URL，避免泄露到 .git/config、进程参数或任务日志。
func BuildRepoURL(platform, owner, repo string) (string, error) {
	ph, ok := PlatformHosts[platform]
	if !ok {
		return "", fmt.Errorf("不支持的平台: %s", platform)
	}
	owner = strings.Trim(owner, "/ ")
	repo = strings.Trim(strings.TrimSuffix(strings.TrimSpace(repo), ".git"), "/")
	if owner == "" || repo == "" {
		return "", fmt.Errorf("仓库 owner / repo 不能为空")
	}
	escapePath := func(value string) (string, error) {
		parts := strings.Split(value, "/")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" || part == "." || part == ".." {
				return "", fmt.Errorf("仓库路径无效: %s", value)
			}
			out = append(out, url.PathEscape(part))
		}
		return strings.Join(out, "/"), nil
	}
	escapedOwner, err := escapePath(owner)
	if err != nil {
		return "", err
	}
	escapedRepo, err := escapePath(repo)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s/%s/%s.git", ph.Host, escapedOwner, escapedRepo), nil
}

// PlatformDisplayName 返回平台展示名（用于日志）
func PlatformDisplayName(platform string) string {
	switch platform {
	case db.PlatformGitHub:
		return "GitHub"
	case db.PlatformGitCode:
		return "GitCode"
	case db.PlatformGitee:
		return "Gitee"
	default:
		return platform
	}
}

func report(progress ProgressFunc, phase string, percent int, message string) {
	if progress != nil {
		progress(phase, percent, message)
	}
}

func runGitContext(ctx context.Context, dir string, log Logger, env []string, args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if output.Len() > 0 {
		for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
			if line != "" {
				log("GIT", line)
			}
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func gitOutputContext(ctx context.Context, dir string, env []string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return out, err
}

// credentialEnv 创建一次性 GIT_ASKPASS，用户名和 Token 只通过子进程环境传递。
func credentialEnv(user, token string) ([]string, func(), error) {
	dir, err := os.MkdirTemp("", "repoark-askpass-")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	script := filepath.Join(dir, "askpass.sh")
	content := "#!/bin/sh\ncase \"$1\" in\n  *Username*) printf '%s\\n' \"$REPOARK_GIT_USERNAME\" ;;\n  *) printf '%s\\n' \"$REPOARK_GIT_TOKEN\" ;;\nesac\n"
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		cleanup()
		return nil, nil, err
	}
	return []string{
		"GIT_ASKPASS=" + script,
		"GIT_TERMINAL_PROMPT=0",
		"REPOARK_GIT_USERNAME=" + user,
		"REPOARK_GIT_TOKEN=" + token,
	}, cleanup, nil
}

// ValidBranchName 校验 Git 分支名称。
func ValidBranchName(branch string) bool {
	return exec.Command("git", "check-ref-format", "--branch", branch).Run() == nil
}

type sourceSpec struct {
	display  string
	resolved string
	dest     string
}

type sourceStats struct {
	files int64
	bytes int64
}

func sourceDestination(source string) string {
	base := filepath.Base(filepath.Clean(source))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "root"
	}
	return base
}

func resolveSources(ctx context.Context, cfg Config, excluded os.FileInfo, log Logger) ([]sourceSpec, error) {
	sources := make([]sourceSpec, 0, len(cfg.BackupSources))
	for _, source := range cfg.BackupSources {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		resolved := source
		if cfg.HostRoot != "" {
			resolved = filepath.Join(cfg.HostRoot, source)
		}
		info, err := os.Lstat(resolved)
		if err != nil {
			log("WARNING", "源路径不存在或无法读取: "+source)
			continue
		}
		if excluded != nil && os.SameFile(info, excluded) {
			log("WARNING", "已跳过临时工作目录，不能把它自身作为备份源: "+source)
			continue
		}
		sources = append(sources, sourceSpec{
			display:  source,
			resolved: resolved,
			dest:     filepath.ToSlash(filepath.Join(cfg.ServerName, sourceDestination(source))),
		})
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("没有找到可备份的有效源路径，请检查路径和宿主机根路径映射")
	}
	return sources, nil
}

func walkSource(ctx context.Context, source sourceSpec, excluded os.FileInfo, visit func(string, string, os.FileInfo) error) error {
	rootInfo, err := os.Lstat(source.resolved)
	if err != nil {
		return err
	}
	if !rootInfo.IsDir() {
		return visit(source.resolved, source.dest, rootInfo)
	}
	return filepath.Walk(source.resolved, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if excluded != nil && info.IsDir() && os.SameFile(info, excluded) {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(source.resolved, path)
		if err != nil {
			return err
		}
		dest := filepath.ToSlash(filepath.Join(source.dest, rel))
		return visit(path, dest, info)
	})
}

func scanSources(ctx context.Context, sources []sourceSpec, excluded os.FileInfo, log Logger, progress ProgressFunc) (sourceStats, error) {
	var stats sourceStats
	lastReport := time.Now()
	for _, source := range sources {
		err := walkSource(ctx, source, excluded, func(path, _ string, info os.FileInfo) error {
			switch {
			case info.Mode().IsRegular():
				stats.files++
				stats.bytes += info.Size()
			case info.Mode()&os.ModeSymlink != 0:
				target, err := os.Readlink(path)
				if err != nil {
					return err
				}
				stats.files++
				stats.bytes += int64(len(target))
			default:
				log("WARNING", "已跳过不受 Git 支持的特殊文件: "+path)
			}
			if time.Since(lastReport) >= 500*time.Millisecond {
				report(progress, "scanning", 12, fmt.Sprintf("正在扫描备份源：已发现 %d 个文件，共 %s", stats.files, humanBytes(stats.bytes)))
				lastReport = time.Now()
			}
			return nil
		})
		if err != nil {
			return stats, fmt.Errorf("扫描备份源 %s 失败: %w", source.display, err)
		}
	}
	if stats.files == 0 {
		return stats, fmt.Errorf("备份源中没有可提交的普通文件或符号链接")
	}
	return stats, nil
}

var writingObjectsPattern = regexp.MustCompile(`Writing objects:\s+(\d+)%`)

func splitGitProgress(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for index, value := range data {
		if value == '\n' || value == '\r' {
			if index == 0 {
				return 1, nil, nil
			}
			return index + 1, data[:index], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func runGitPushContext(ctx context.Context, repoDir string, log Logger, env []string, branch string, progress ProgressFunc, attempt int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "git", "--git-dir", repoDir, "push", "--progress", "origin", "refs/heads/"+branch+":refs/heads/"+branch)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stderr)
	scanner.Split(splitGitProgress)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		log("GIT", line)
		match := writingObjectsPattern.FindStringSubmatch(line)
		if len(match) == 2 {
			written, parseErr := strconv.Atoi(match[1])
			if parseErr == nil {
				overall := 92 + written*7/100
				if overall > 99 {
					overall = 99
				}
				report(progress, "pushing", overall, fmt.Sprintf("正在第 %d 次推送：Git 对象已上传 %d%%", attempt, written))
			}
		}
	}
	scanErr := scanner.Err()
	waitErr := cmd.Wait()
	if stdout.Len() > 0 {
		for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
			if line != "" {
				log("GIT", line)
			}
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if scanErr != nil {
		return scanErr
	}
	return waitErr
}

type importTracker struct {
	totalBytes int64
	totalFiles int64
	doneBytes  int64
	doneFiles  int64
	start      int
	end        int
	progress   ProgressFunc
	lastReport time.Time
}

func (t *importTracker) advance(bytes int64, fileDone bool, path string) {
	t.doneBytes += bytes
	if fileDone {
		t.doneFiles++
	}
	if !fileDone && time.Since(t.lastReport) < 250*time.Millisecond {
		return
	}
	fraction := float64(t.doneFiles) / float64(t.totalFiles)
	if t.totalBytes > 0 {
		fraction = float64(t.doneBytes) / float64(t.totalBytes)
	}
	if fraction > 1 {
		fraction = 1
	}
	percent := t.start + int(fraction*float64(t.end-t.start))
	message := fmt.Sprintf("正在生成 Git 对象：%s（%d/%d 个文件）", path, t.doneFiles, t.totalFiles)
	if t.totalBytes > 0 {
		message = fmt.Sprintf("正在生成 Git 对象：%s（%s/%s）", path, humanBytes(t.doneBytes), humanBytes(t.totalBytes))
	}
	report(t.progress, "packing", percent, message)
	t.lastReport = time.Now()
}

func humanBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	div, exp := int64(unit), 0
	for n := value / unit; n >= unit && exp < 5; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(value)/float64(div), "KMGTPE"[exp])
}

func fastImportPath(path string) string {
	path = filepath.ToSlash(path)
	safe := path != ""
	for i := 0; i < len(path); i++ {
		if path[i] <= 0x20 || path[i] >= 0x7f || path[i] == '"' || path[i] == '\\' {
			safe = false
			break
		}
	}
	if safe {
		return path
	}
	var out strings.Builder
	out.WriteByte('"')
	for i := 0; i < len(path); i++ {
		c := path[i]
		switch c {
		case '"', '\\':
			out.WriteByte('\\')
			out.WriteByte(c)
		case '\n':
			out.WriteString("\\n")
		case '\r':
			out.WriteString("\\r")
		case '\t':
			out.WriteString("\\t")
		default:
			if c < 0x20 || c >= 0x7f {
				fmt.Fprintf(&out, "\\%03o", c)
			} else {
				out.WriteByte(c)
			}
		}
	}
	out.WriteByte('"')
	return out.String()
}

func safeIdentity(value string) string {
	value = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '<' || r == '>' {
			return -1
		}
		return r
	}, value)
	if strings.TrimSpace(value) == "" {
		return "RepoArk"
	}
	return strings.TrimSpace(value)
}

func timezoneOffset(now time.Time) string {
	_, offset := now.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	return fmt.Sprintf("%s%02d%02d", sign, offset/3600, offset%3600/60)
}

func writeRegularFile(ctx context.Context, writer *bufio.Writer, path string, size int64, tracker *importTracker, display string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	remaining := size
	buffer := make([]byte, 1024*1024)
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		chunk := int64(len(buffer))
		if remaining < chunk {
			chunk = remaining
		}
		n, readErr := io.ReadFull(file, buffer[:chunk])
		if n > 0 {
			if _, err := writer.Write(buffer[:n]); err != nil {
				return err
			}
			remaining -= int64(n)
			tracker.advance(int64(n), false, display)
		}
		if readErr != nil {
			return fmt.Errorf("文件在备份过程中发生变化: %w", readErr)
		}
	}
	tracker.advance(0, true, display)
	return nil
}

func importSnapshot(ctx context.Context, repoDir string, cfg Config, sources []sourceSpec, excluded os.FileInfo, stats sourceStats, hasBase bool, progress ProgressFunc, startPercent, endPercent int) error {
	cmd := exec.CommandContext(ctx, "git", "--git-dir", repoDir, "fast-import", "--quiet", "--force")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		return err
	}

	tracker := &importTracker{
		totalBytes: stats.bytes,
		totalFiles: stats.files,
		start:      startPercent,
		end:        endPercent,
		progress:   progress,
	}
	writer := bufio.NewWriterSize(stdin, 1024*1024)
	now := time.Now()
	branchRef := "refs/heads/" + cfg.Branch
	baseRef := "refs/remotes/origin/" + cfg.Branch
	name := safeIdentity(cfg.GitUser)
	email := name + "@users.noreply." + PlatformHosts[cfg.Platform].Host
	commitMessage := fmt.Sprintf("Backup update: %s - %s", cfg.ServerName, now.Format("2006-01-02 15:04:05"))

	writeErr := func() error {
		fmt.Fprintf(writer, "commit %s\n", branchRef)
		fmt.Fprintf(writer, "author %s <%s> %d %s\n", name, email, now.Unix(), timezoneOffset(now))
		fmt.Fprintf(writer, "committer %s <%s> %d %s\n", name, email, now.Unix(), timezoneOffset(now))
		fmt.Fprintf(writer, "data %d\n%s\n", len(commitMessage), commitMessage)
		if hasBase {
			fmt.Fprintf(writer, "from %s\n", baseRef)
		}
		fmt.Fprintf(writer, "D %s\n", fastImportPath(cfg.ServerName))

		for _, source := range sources {
			err := walkSource(ctx, source, excluded, func(path, dest string, info os.FileInfo) error {
				mode := "100644"
				var symlinkData []byte
				switch {
				case info.Mode().IsRegular():
					if info.Mode().Perm()&0o111 != 0 {
						mode = "100755"
					}
				case info.Mode()&os.ModeSymlink != 0:
					mode = "120000"
					target, err := os.Readlink(path)
					if err != nil {
						return err
					}
					symlinkData = []byte(target)
				default:
					return nil
				}
				fmt.Fprintf(writer, "M %s inline %s\n", mode, fastImportPath(dest))
				if symlinkData != nil {
					fmt.Fprintf(writer, "data %d\n", len(symlinkData))
					if _, err := writer.Write(symlinkData); err != nil {
						return err
					}
					tracker.advance(int64(len(symlinkData)), true, dest)
				} else {
					fmt.Fprintf(writer, "data %d\n", info.Size())
					if err := writeRegularFile(ctx, writer, path, info.Size(), tracker, dest); err != nil {
						return err
					}
				}
				return writer.WriteByte('\n')
			})
			if err != nil {
				return fmt.Errorf("读取备份源 %s 失败: %w", source.display, err)
			}
		}
		report(progress, "committing", endPercent, "正在创建备份提交")
		_, err := writer.WriteString("done\n")
		return err
	}()
	flushErr := writer.Flush()
	closeErr := stdin.Close()
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	for _, candidate := range []error{writeErr, flushErr, closeErr, waitErr} {
		if candidate != nil {
			if output.Len() > 0 {
				return fmt.Errorf("生成 Git 提交失败: %w: %s", candidate, strings.TrimSpace(output.String()))
			}
			return fmt.Errorf("生成 Git 提交失败: %w", candidate)
		}
	}
	return nil
}

type remoteState struct {
	hasTarget      bool
	hasAny         bool
	supportsFilter bool
}

func inspectRemote(ctx context.Context, repoDir, branch string, authEnv []string) (remoteState, error) {
	probeEnv := append([]string{}, authEnv...)
	probeEnv = append(probeEnv, "GIT_TRACE_PACKET=1", "LC_ALL=C", "LANG=C")
	out, err := gitOutputContext(ctx, "", probeEnv, "-c", "protocol.version=2", "--git-dir", repoDir, "ls-remote", "--heads", "origin")
	if err != nil {
		return remoteState{}, fmt.Errorf("读取远程仓库分支失败: %w: %s", err, strings.TrimSpace(string(out)))
	}
	target := "refs/heads/" + branch
	state := remoteState{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.Contains(line, "fetch=") && strings.Contains(line, " filter") {
			state.supportsFilter = true
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || !strings.HasPrefix(fields[1], "refs/heads/") {
			continue
		}
		state.hasAny = true
		if fields[1] == target {
			state.hasTarget = true
		}
	}
	return state, nil
}

// fetchRemoteMetadata 只下载远端 commit/tree 元数据，不下载任何历史文件 blob。
// 先由 inspectRemote 验证服务端支持 partial clone filter，避免服务端忽略参数后退化为完整下载。
func fetchRemoteMetadata(ctx context.Context, repoDir, branch string, authEnv []string, log Logger) error {
	fetchEnv := append([]string{}, authEnv...)
	fetchEnv = append(fetchEnv, "LC_ALL=C", "LANG=C")
	return runGitContext(
		ctx,
		"",
		log,
		fetchEnv,
		"-c", "protocol.version=2",
		"--git-dir", repoDir,
		"fetch",
		"--no-tags",
		"--depth=1",
		"--filter=blob:none",
		"origin",
		"+refs/heads/"+branch+":refs/remotes/origin/"+branch,
	)
}

func sameTree(ctx context.Context, repoDir, branch string) (bool, error) {
	local, err := gitOutputContext(ctx, "", nil, "--git-dir", repoDir, "rev-parse", "refs/heads/"+branch+"^{tree}")
	if err != nil {
		return false, err
	}
	remote, err := gitOutputContext(ctx, "", nil, "--git-dir", repoDir, "rev-parse", "refs/remotes/origin/"+branch+"^{tree}")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(local)) == strings.TrimSpace(string(remote)), nil
}

func waitContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Run 保留同步调用入口，测试和其他调用方无需关心取消与进度。
func Run(cfg Config, log Logger) error {
	return RunContext(context.Background(), cfg, log, nil)
}

// RunContext 直接把源文件流式写入临时 Git 对象库并推送，不再创建一份完整工作区副本。
func RunContext(ctx context.Context, cfg Config, log Logger, progress ProgressFunc) error {
	report(progress, "preparing", 2, "正在检查备份配置")
	if err := ctx.Err(); err != nil {
		return err
	}
	if cfg.Platform == "" {
		cfg.Platform = db.PlatformGitHub
	}
	cfg.ServerName = strings.TrimSpace(cfg.ServerName)
	if err := db.ValidateNodeName(cfg.ServerName); err != nil {
		return err
	}
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	if !ValidBranchName(cfg.Branch) {
		return fmt.Errorf("分支名称无效: %s", cfg.Branch)
	}
	if strings.TrimSpace(cfg.BackupDir) == "" {
		return fmt.Errorf("临时工作目录未配置")
	}
	if cfg.GitUser == "" || cfg.GitToken == "" || cfg.RepoName == "" {
		return fmt.Errorf("%s 仓库信息不完整（需要 user/token/repo）", PlatformDisplayName(cfg.Platform))
	}
	validSources := make([]string, 0, len(cfg.BackupSources))
	for _, source := range cfg.BackupSources {
		if source = strings.TrimSpace(source); source != "" {
			validSources = append(validSources, source)
		}
	}
	if len(validSources) == 0 {
		return fmt.Errorf("未配置备份源路径；临时工作目录不是备份源路径")
	}
	cfg.BackupSources = validSources

	repoURL := strings.TrimSpace(cfg.RemoteURL)
	if repoURL == "" {
		var err error
		repoURL, err = BuildRepoURL(cfg.Platform, cfg.GitUser, cfg.RepoName)
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(cfg.BackupDir, 0o755); err != nil {
		return fmt.Errorf("创建临时工作目录失败: %w", err)
	}
	excluded, _ := os.Stat(cfg.BackupDir)
	tempDir, err := os.MkdirTemp(cfg.BackupDir, ".repoark-run-")
	if err != nil {
		return fmt.Errorf("创建临时任务目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)
	repoDir := filepath.Join(tempDir, "repo.git")

	report(progress, "preparing", 5, "正在初始化临时 Git 对象库")
	if err := runGitContext(ctx, "", log, nil, "init", "--bare", repoDir); err != nil {
		return fmt.Errorf("初始化临时 Git 仓库失败: %w", err)
	}
	if err := runGitContext(ctx, "", log, nil, "--git-dir", repoDir, "remote", "add", "origin", repoURL); err != nil {
		return fmt.Errorf("配置远程仓库失败: %w", err)
	}
	authEnv, cleanupCredentials, err := credentialEnv(cfg.GitUser, cfg.GitToken)
	if err != nil {
		return fmt.Errorf("初始化 Git 凭据失败: %w", err)
	}
	defer cleanupCredentials()

	report(progress, "scanning", 10, "正在扫描备份源文件")
	sources, err := resolveSources(ctx, cfg, excluded, log)
	if err != nil {
		return err
	}
	stats, err := scanSources(ctx, sources, excluded, log, progress)
	if err != nil {
		return err
	}
	log("INFO", fmt.Sprintf("扫描完成：%d 个文件，共 %s；将直接生成 Git 对象，不创建完整文件副本", stats.files, humanBytes(stats.bytes)))
	report(progress, "connecting", 18, "正在连接远程仓库")

	const maxRetries = 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		state, err := inspectRemote(ctx, repoDir, cfg.Branch, authEnv)
		if err != nil {
			return err
		}
		if !state.hasTarget && state.hasAny {
			return fmt.Errorf("远程仓库不存在分支 %s，请检查分支配置", cfg.Branch)
		}
		if state.hasTarget {
			if !state.supportsFilter {
				return fmt.Errorf("远程 Git 服务不支持仅获取仓库元数据（blob:none），已停止备份以避免下载远端备份文件")
			}
			percent := 25
			if attempt > 1 {
				percent = 90
			}
			report(progress, "fetching", percent, "正在获取远程提交和目录树元数据（不下载文件内容）")
			if err := fetchRemoteMetadata(ctx, repoDir, cfg.Branch, authEnv, log); err != nil {
				return fmt.Errorf("获取远程仓库元数据失败: %w", err)
			}
		} else {
			log("INFO", "远程仓库为空，将创建初始分支 "+cfg.Branch)
		}

		startPercent, endPercent := 35, 75
		if attempt > 1 {
			startPercent, endPercent = 91, 96
		}
		report(progress, "packing", startPercent, "正在从源目录直接生成 Git 对象")
		if err := importSnapshot(ctx, repoDir, cfg, sources, excluded, stats, state.hasTarget, progress, startPercent, endPercent); err != nil {
			return err
		}
		if state.hasTarget {
			equal, err := sameTree(ctx, repoDir, cfg.Branch)
			if err != nil {
				return fmt.Errorf("检查仓库变更失败: %w", err)
			}
			if equal {
				log("INFO", "没有发现新的更改，无需推送")
				report(progress, "completed", 100, "备份内容没有变化")
				return nil
			}
		}

		pushPercent := 90 + attempt*2
		if pushPercent > 98 {
			pushPercent = 98
		}
		report(progress, "pushing", pushPercent, fmt.Sprintf("正在第 %d 次推送提交到 %s", attempt, PlatformDisplayName(cfg.Platform)))
		log("INFO", fmt.Sprintf("第 %d 次尝试推送到远程仓库...", attempt))
		err = runGitPushContext(ctx, repoDir, log, authEnv, cfg.Branch, progress, attempt)
		if err == nil {
			log("SUCCESS", "备份完成！"+cfg.ServerName+" 的文件已成功推送到 "+PlatformDisplayName(cfg.Platform)+" 仓库")
			report(progress, "completed", 100, "备份已成功推送到远程仓库")
			return nil
		}
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			return context.Canceled
		}
		if attempt == maxRetries {
			return fmt.Errorf("推送失败次数达到上限，请检查仓库状态")
		}
		log("WARNING", "推送失败，5 秒后重新同步远程状态并重试...")
		report(progress, "retrying", pushPercent, "远程仓库发生变化，等待重新同步")
		if err := waitContext(ctx, 5*time.Second); err != nil {
			return err
		}
	}
	return fmt.Errorf("推送失败")
}
