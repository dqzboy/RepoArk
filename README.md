# RepoArk
在服务器中运行服务，在WEB管理界面设置备份文件和定时任务，实现定时备份数据到GitHub仓库


<br/>
<table>
    <tr>
      <td width="50%" align="center"><b>仪表盘</b></td>
      <td width="50%" align="center"><b>备份设置</b></td>
    </tr>
    <tr>
        <td width="50%" align="center"><img src="https://cdn.jsdelivr.net/gh/dqzboy/Images/picture/git_bak_sync-01.png?raw=true"></td>
        <td width="50%" align="center"><img src="https://cdn.jsdelivr.net/gh/dqzboy/Images/picture/git_bak_sync-02.png?raw=true"></td>
    </tr>
</table>

## 运行方式

### 1. 一键构建并运行

```bash
./scripts/build.sh
./git-backup-server 
```

首次启动会在 `./data/app.db` 创建 SQLite 库并写入默认配置（二进制运行时数据目录为当前工作目录下的 `data/`）。
默认管理员账号：**admin / admin**（请尽快在「备份配置」中修改）。

### 2. 开发模式

```bash
cd server && go run .
cd web && npm install && npm run dev
```

打开 http://localhost:5173 进行前端开发调试。

### 3. Docker 运行

- 支持 `linux/amd64` 与 `linux/arm64`，直接拉取运行即可：

```bash
# 拉取最新镜像
docker pull ghcr.io/dqzboy/repoark:latest

# 运行（数据持久化到宿主机 ./data）
docker run -d --name repoark-server \
  -p 8080:8080 \
  -v "$(pwd)/data:/app/data" \
  ghcr.io/dqzboy/repoark:latest
```

启动后访问 http://localhost:8080 。默认管理员账号：**admin / admin**。

### 4. Docker Compose 运行

项目根目录已提供 `docker-compose.yml`，直接一键启动：

```bash
docker compose up -d
```

停止与清理：

```bash
docker compose down
```

如需指定版本，编辑 `docker-compose.yml` 中的 `image` 标签（如 `:v1.0.0`）。数据同样持久化在宿主机 `./data` 目录。

#### 备份宿主机上的文件 / 文件夹（Docker 专属）

容器有独立的文件系统隔离，默认看不到宿主机的 `/etc`、`/var/www` 等目录。要像二进制直接运行那样在「备份配置」里填写**真实路径**（如 `/etc/nginx/conf.d`），需要把宿主机根目录挂进容器并启用路径映射：


> 提示：`- /:/host:ro` 以只读方式把整棵宿主机目录树挂到容器 `/host`，一次覆盖所有分散目录，无需逐个挂载；只读挂载可避免备份程序误改宿主机文件。

### 5. 使用流程

1. 登录后进入「备份配置」，填写 GitHub 用户名、Token、仓库名、分支，以及要备份的源路径（如 `/etc/passwd`、`/etc/nginx/conf.d`）。
2. 进入「执行备份」点击「开始备份」，后端会按当前配置执行备份，页面实时显示日志。
3. 在「任务历史」可查看每次备份的结果与完整日志。

## 前提条件：

<details open>
<summary>点击展开 ...</summary>

<div align="center">

**1、** 创建GitHub仓库，设置为私有

<table>
    <tr>
        <td width="50%" align="center"><img src="https://github.com/user-attachments/assets/f4b750c3-b4cd-48e0-8bc3-2313d45726dd"?raw=true"></td>
    </tr>
</table>


**2、** 创建GitHubToken，给个pull、push权限即可
<table>
    <tr>
        <td width="50%" align="center"><img src="https://github.com/user-attachments/assets/fc51040f-a7ea-4b9e-bc7e-c35469849674"?raw=true"></td>
    </tr>
</table>
<table>
    <tr>
        <td width="50%" align="center"><img src="https://github.com/user-attachments/assets/bf54121f-ccd7-4058-84fb-25f3a526e679"?raw=true"></td>
    </tr>
</table>
<table>
    <tr>
        <td width="50%" align="center"><img src="https://github.com/user-attachments/assets/1e38b9d1-5da3-4056-b967-a5fbdaa93e39"?raw=true"></td>
    </tr>
</table>

</div>

</details>



**注意**：把**Toekn**保留下来，只会出现一次。在后台配置时需要使用到！



## 💌 推广

<table>
  <thead>
    <tr>
      <th width="50%" align="center">描述信息</th>
      <th width="50%" align="center">图文介绍</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td width="50%" align="left">
        <a href="https://docker-proxy-desc.vercel.app/racknerd.html" target="_blank">提供高性价比的海外VPS，支持多种操作系统，适合搭建Docker代理服务。</a>
      </td>
      <td width="50%" align="center">
        <a href="https://docker-proxy-desc.vercel.app/racknerd.html" target="_blank">
          <img src="https://cdn.jsdelivr.net/gh/dqzboy/Images/dqzboy-proxy/Image_2025-07-07_16-14-49.png?raw=true" alt="RackNerd" width="200" height="150">
        </a>
      </td>
    </tr>
    <tr>
      <td width="50%" align="left">
        <a href="https://docker-proxy-desc.vercel.app/cloudcone.html" target="_blank">CloudCone 提供灵活的云服务器方案，支持按需付费，适合个人和企业用户。</a>
      </td>
      <td width="50%" align="center">
        <a href="https://docker-proxy-desc.vercel.app/cloudcone.html" target="_blank">
          <img src="https://cdn.jsdelivr.net/gh/dqzboy/Images/dqzboy-proxy/111.png?raw=true" alt="CloudCone" width="200" height="150">
        </a>
      </td>
    </tr>
  </tbody>
</table>

##### *Telegram Bot: [点击联系](https://t.me/WiseAidBot) ｜ E-Mail: support@dqzboy.com*
**仅接受长期稳定运营，信誉良好的商家*
