package git

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"git-backup-web/server/internal/db"
)

// Config 备份执行配置
type Config struct {
	Platform     string   // github | gitcode | gitee（影响 repo URL 拼接）
	GitUser      string   // 用户名 / 命名空间
	GitToken     string
	RepoName     string
	Branch       string
	BackupDir    string
	ServerName   string
	BackupSources []string
	HostRoot     string
}

// Logger 日志回调：level 取值 INFO/SUCCESS/WARNING/ERROR/GIT
type Logger func(level, msg string)

// PlatformHost 不同平台的 git host 与 owner 段拼接规则
type platformHost struct {
	Host       string // 远端主机名（不带协议）
	UserInPath bool   // true: gitcode/gitee 把 owner 放在 path 第一段
}

// PlatformHosts 各平台的远端 host 配置
var PlatformHosts = map[string]platformHost{
	db.PlatformGitHub:  {Host: "github.com", UserInPath: true},
	db.PlatformGitCode: {Host: "gitcode.net", UserInPath: true},
	db.PlatformGitee:   {Host: "gitee.com", UserInPath: true},
}

// BuildRepoURL 按平台拼接 https://token@host/owner/repo.git
func BuildRepoURL(platform, token, owner, repo string) (string, error) {
	ph, ok := PlatformHosts[platform]
	if !ok {
		return "", fmt.Errorf("不支持的平台: %s", platform)
	}
	if owner == "" || repo == "" {
		return "", fmt.Errorf("仓库 owner / repo 不能为空")
	}
	return fmt.Sprintf("https://%s@%s/%s/%s.git", token, ph.Host, owner, repo), nil
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
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
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

	repoURL, err := BuildRepoURL(cfg.Platform, cfg.GitToken, cfg.GitUser, cfg.RepoName)
	if err != nil {
		return err
	}
	backupDir := cfg.BackupDir
	serverBackupDir := filepath.Join(backupDir, cfg.ServerName)

	// 1. 克隆仓库（如果不存在）
	if _, err := os.Stat(filepath.Join(backupDir, ".git")); err != nil {
		log("INFO", "备份目录不是 git 仓库，正在克隆...")
		if err := runGit("", log, "clone", "-b", cfg.Branch, repoURL, backupDir); err != nil {
			return fmt.Errorf("克隆仓库失败: %w", err)
		}
		log("SUCCESS", "仓库克隆成功")
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
	_ = runGit(backupDir, log, "fetch", "origin")
	_ = runGit(backupDir, log, "reset", "--hard", "origin/"+cfg.Branch)
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
		_ = runGit(backupDir, log, "fetch", "origin")
		if err := runGit(backupDir, log, "rebase", "origin/"+cfg.Branch); err != nil {
			log("WARNING", "变基失败，正在中止并重置...")
			_ = runGit(backupDir, log, "rebase", "--abort")
			cleanup()
			_ = runGit(backupDir, log, "fetch", "origin")
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
		if err := runGit(backupDir, log, "push", "origin", cfg.Branch); err == nil {
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
