# 同步上游更新指南

## 分支和 Remote 约定

| 名称 | 用途 |
| --- | --- |
| `main` | 本仓库主分支，不直接承载日常开发 |
| `stable` | `111.228.5.198` 当前稳定版本 |
| `develop` | 日常开发、定制功能和上游更新的集成分支 |
| `origin` | `https://github.com/tangqiaojian/new-api.git` |
| `upstream` | `https://github.com/QuantumNous/new-api.git` |

## 将官方主版本合入开发分支

```bash
git switch develop
git fetch --prune upstream
git merge --no-ff upstream/main

# 解决冲突并完成测试后
git push origin develop
```

不要直接把 `upstream/main` 合入 `stable`。只有在生产部署完成并验证后，才按实际运行的提交更新 `stable`。

## 查看差异

```bash
# 官方尚未进入 develop 的提交
git log develop..upstream/main --oneline

# develop 的定制提交
git log upstream/main..develop --oneline

# 文件级差异
git diff upstream/main...develop
```

仓库只保留 `main`、`stable`、`develop` 三个长期分支。临时开发应在本地完成并合入 `develop`，不要长期保留远端杂项分支。
