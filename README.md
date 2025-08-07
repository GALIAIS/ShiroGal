# ShiroGal - Galgame 游戏资源客户端

ShiroGal 是一个优雅的桌面应用程序，方便用户浏览、搜索 Galgame（美少女游戏）信息。基于轻量级本地数据库缓存，提供流畅的离线体验，并通过云端实时同步保持数据最新。

## ✨ 主要功能
- **实时数据同步**：启动时自动从云端同步最新游戏数据。
- **快速本地搜索**：支持游戏标题（日文/中文）和品牌的即时搜索。
- **详细游戏信息**：包括发行日期、剧情简介、封面、预览图、标签等。
- **跨平台支持**：基于 Wails 框架，支持 Windows 和 macOS。
- **现代化界面**：简洁美观，提供流畅浏览体验。

## 🚀 快速开始（普通用户）
1. 访问 [GitHub Releases](https://github.com/GALIAIS/ShiroGal/releases)。
2. 下载最新版本文件。
3. 下载后直接运行。

## 🛠️ 开发与构建（开发者）

### 先决条件
- Go（1.22+）
- Node.js（20.x LTS+）
- Wails CLI

### 本地开发
1. 克隆仓库：
   ```bash
   git clone https://github.com/GALIAIS/ShiroGal.git
   cd ShiroGal
   ```
2. 安装前端依赖：
   ```bash
   cd frontend
   npm install
   cd ..
   ```
3. 运行开发模式：
   ```bash
   wails dev -ldflags="-X main.dataServiceURL=https://api.example.com/api/v1 -X main.publicKey=XXXXXX -X main.privateKey=XXXXXXXXXXXXXXXXXXXXXXXX"
   ```

### 生产构建
```bash
wails build -ldflags="-X main.dataServiceURL=https://api.example.com/api/v1 -X main.publicKey=XXXXXX -X main.privateKey=XXXXXXXXXXXXXXXXXXXXXXXX" -clean -upx -webview2 embed
```
构建产物位于 `build/bin` 目录。

### 预览图
![img.png](img.png)
![img_1.png](img_1.png)

## 🔧 技术栈
- **框架**：Wails v2
- **后端**：Go
- **前端**：Svelte / Vue / React
- **本地数据库**：SQLite
- **云端数据库**：TiDB Cloud
- **API 服务**：TiDB Cloud Data Service
- **CI/CD**：GitHub Actions