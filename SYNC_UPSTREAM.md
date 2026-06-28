# 同步上游更新指南

## Remote 配置

| 名称 | 地址 | 说明 |
|------|------|------|
| `origin` | `https://github.com/tangqiaojian/new-api.git` | 你的 GitHub 仓库（默认推送目标） |
| `upstream` | `https://github.com/QuantumNous/new-api.git` | 上游原项目 |

## 同步上游更新

当原项目有新的功能更新时，执行以下命令合并到你的仓库：

```bash
# 1. 拉取上游最新代码
git fetch upstream

# 2. 合并到本地 main 分支
git merge upstream/main

# 3. 如有冲突，解决后提交
# git add .
# git commit -m "merge: resolve conflicts with upstream"

# 4. 推送到你的 GitHub 仓库
git push origin main
```

## 日常开发推送

```bash
git push origin main
```

## 查看与上游的差异

```bash
# 查看本地与上游的提交差异
git log HEAD..upstream/main --oneline

# 查看文件差异
git diff upstream/main
```
