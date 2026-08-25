package git

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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

// DetectServerName 自动探测服务器标识（优先取第一个内网 IP）
func DetectServerName() string {
	if out, err := exec.Command("hostname", "-I").Output(); err == nil {
		fields := strings.Fields(string(out))
		if len(fields) > 0 {
			return fields[0]
		}
	}
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown-server"
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

func runGit(dir string, log Logger, args ...string) error {
	return runGitEnv(dir, log, nil, args...)
}

func runGitEnv(dir string, log Logger, env []string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if buf.Len() > 0 {
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			if line != "" {
				log("GIT", line)
			}
		}
	}
	return err
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

// hasStagedChanges git diff --cached --quiet 退出码非 0 表示存在暂存变更
func hasStagedChanges(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "diff", "--cached", "--quiet")
	return cmd.Run() != nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else {
			if err := copyFile(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copySources(sources []string, hostRoot, serverBackupDir string, log Logger) {
	for _, src := range sources {
		src = strings.TrimSpace(src)
		if src == "" {
			continue
		}
		// Docker 部署时把宿主机根挂到容器的 hostRoot（如 /host），此处自动拼成容器内真实路径
		resolved := src
		if hostRoot != "" {
			resolved = filepath.Join(hostRoot, src)
		}
		if _, err := os.Stat(resolved); err == nil {
			dst := filepath.Join(serverBackupDir, filepath.Base(src))
			if err := copyPath(resolved, dst); err != nil {
				log("WARNING", fmt.Sprintf("复制失败: %s -> %v", src, err))
			} else {
				log("INFO", "成功复制: "+src)
			}
		} else {
			log("WARNING", "源路径不存在: "+src)
		}
	}
}

// Run 执行一次完整备份，逻辑等价于原 git_sync_backup.sh
func Run(cfg Config, log Logger) error {
	if cfg.Platform == "" {
		cfg.Platform = db.PlatformGitHub
	}
	if cfg.ServerName == "" {
		cfg.ServerName = DetectServerName()
	}
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}
	if cfg.BackupDir == "" {
		return fmt.Errorf("备份目录未配置")
	}
	if cfg.GitUser == "" || cfg.GitToken == "" || cfg.RepoName == "" {
		return fmt.Errorf("%s 仓库信息不完整（需要 user/token/repo）", PlatformDisplayName(cfg.Platform))
	}

	repoURL := strings.TrimSpace(cfg.RemoteURL)
	if repoURL == "" {
		var err error
		repoURL, err = BuildRepoURL(cfg.Platform, cfg.GitUser, cfg.RepoName)
		if err != nil {
			return err
		}
	}
	authEnv, cleanupCredentials, err := credentialEnv(cfg.GitUser, cfg.GitToken)
	if err != nil {
		return fmt.Errorf("初始化 Git 凭据失败: %w", err)
	}
	defer cleanupCredentials()
	runRemoteGit := func(dir string, args ...string) error {
		return runGitEnv(dir, log, authEnv, args...)
	}
	backupDir := cfg.BackupDir
	serverBackupDir := filepath.Join(backupDir, cfg.ServerName)

	// 1. 克隆仓库（如果不存在）
	if _, err := os.Stat(filepath.Join(backupDir, ".git")); err != nil {
		log("INFO", "备份目录不是 git 仓库，正在克隆...")
		if err := runRemoteGit("", "clone", "-b", cfg.Branch, repoURL, backupDir); err != nil {
			return fmt.Errorf("克隆仓库失败: %w", err)
		}
		log("SUCCESS", "仓库克隆成功")
	}
	// 兼容旧版本已把 Token 写入 origin URL 的仓库，并同步修复 GitCode 旧域名。
	if err := runGit(backupDir, log, "remote", "set-url", "origin", repoURL); err != nil {
		return fmt.Errorf("更新远程仓库地址失败: %w", err)
	}

	// 2. 配置 git 用户信息（用平台提供的 noreply 邮箱作为占位）
	noreply := cfg.GitUser + "@users.noreply." + PlatformHosts[cfg.Platform].Host
	_ = runGit(backupDir, log, "config", "user.name", cfg.GitUser)
	_ = runGit(backupDir, log, "config", "user.email", noreply)

	// 3. 清理并更新到最新
	cleanup := func() {
		_ = runGit(backupDir, log, "reset", "--hard")
		_ = runGit(backupDir, log, "clean", "-fd")
	}
	cleanup()
	_ = runGit(backupDir, log, "config", "core.sparseCheckout", "false")
	_ = os.Remove(filepath.Join(backupDir, ".git", "info", "sparse-checkout"))
	if err := runRemoteGit(backupDir, "fetch", "origin"); err != nil {
		return fmt.Errorf("拉取远程仓库失败: %w", err)
	}
	if err := runGit(backupDir, log, "reset", "--hard", "origin/"+cfg.Branch); err != nil {
		return fmt.Errorf("切换到远程分支 %s 失败: %w", cfg.Branch, err)
	}
	log("SUCCESS", "本地仓库已更新到最新状态")

	// 4. 清理当前服务器的旧备份目录
	if err := os.MkdirAll(serverBackupDir, 0o755); err != nil {
		return fmt.Errorf("创建服务器备份目录失败: %w", err)
	}
	entries, _ := os.ReadDir(serverBackupDir)
	for _, e := range entries {
		_ = os.RemoveAll(filepath.Join(serverBackupDir, e.Name()))
	}
	log("INFO", "已清理当前服务器的旧备份文件")

	// 5. 拷贝新文件
	copySources(cfg.BackupSources, cfg.HostRoot, serverBackupDir, log)

	// 6. 检查变更并提交
	_ = runGit(backupDir, log, "add", cfg.ServerName)
	if !hasStagedChanges(backupDir) {
		log("INFO", "没有发现新的更改，无需备份")
		return nil
	}

	commitMsg := fmt.Sprintf("Backup update: %s - %s", cfg.ServerName, time.Now().Format("2006-01-02 15:04:05"))
	if err := runGit(backupDir, log, "commit", "-m", commitMsg); err != nil {
		return fmt.Errorf("提交失败: %w", err)
	}

	// 7. 推送（带重试与变基）
	const maxRetries = 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log("INFO", fmt.Sprintf("第 %d 次尝试推送到远程仓库...", attempt))
		if err := runRemoteGit(backupDir, "fetch", "origin"); err != nil {
			log("WARNING", "拉取远程更新失败: "+err.Error())
		}
		if err := runGit(backupDir, log, "rebase", "origin/"+cfg.Branch); err != nil {
			log("WARNING", "变基失败，正在中止并重置...")
			_ = runGit(backupDir, log, "rebase", "--abort")
			cleanup()
			_ = runRemoteGit(backupDir, "fetch", "origin")
			_ = runGit(backupDir, log, "reset", "--hard", "origin/"+cfg.Branch)
			// 重新应用本地更改
			entries, _ := os.ReadDir(serverBackupDir)
			for _, e := range entries {
				_ = os.RemoveAll(filepath.Join(serverBackupDir, e.Name()))
			}
			copySources(cfg.BackupSources, cfg.HostRoot, serverBackupDir, log)
			_ = runGit(backupDir, log, "add", cfg.ServerName)
			_ = runGit(backupDir, log, "commit", "-m", commitMsg)
		}
		if err := runRemoteGit(backupDir, "push", "origin", cfg.Branch); err == nil {
			log("SUCCESS", "备份完成！"+cfg.ServerName+" 的文件已成功推送到 "+PlatformDisplayName(cfg.Platform)+" 仓库")
			return nil
		}
		if attempt == maxRetries {
			return fmt.Errorf("推送失败次数达到上限，请检查仓库状态")
		}
		log("WARNING", "推送失败，5 秒后重试...")
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("推送失败")
}
