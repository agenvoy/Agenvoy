# 步驟 1：版本偵測

取得最新語義化標籤，並組出 GitHub compare link 前綴。

**前置條件：** 有 `origin` remote 時，必須先確認遠端 main/master 分支的標籤狀態，才能繼續後續步驟——避免 local tag 落後或漂移於 feature branch 上，導致版本升級基準錯誤。

```bash
# --force 覆蓋遠端已移動的 tag（避免 "would clobber existing tag" 讓本地殘留舊值）
git fetch --tags --force --quiet 2>/dev/null

DEFAULT_BRANCH=""
if git remote get-url origin >/dev/null 2>&1; then
    DEFAULT_BRANCH=$(git ls-remote --symref origin HEAD 2>/dev/null | awk '/^ref:/{sub("refs/heads/", "", $2); print $2}')
    [ -n "$DEFAULT_BRANCH" ] && git fetch --quiet origin "$DEFAULT_BRANCH" 2>/dev/null
fi

if [ -n "$DEFAULT_BRANCH" ] && git rev-parse --verify -q "origin/$DEFAULT_BRANCH" >/dev/null; then
    # 有 remote：LATEST_TAG 必須是 merged 進 origin main/master 的標籤，不採信 local-only 或 feature branch 上的標籤
    LATEST_TAG=$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname --merged "origin/$DEFAULT_BRANCH" | head -1)
else
    # 無 remote（純本地 repo）：回退為 local tag
    LATEST_TAG=$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -1)
fi

if [ -z "$LATEST_TAG" ]; then
    LATEST_TAG="v0.0.0"
fi

# 將 SSH / HTTPS git URL 正規化為 https://github.com/{owner}/{repo}
REMOTE_URL=$(git config --get remote.origin.url \
    | sed -E 's#(git@github\.com:|https://github\.com/)([^/]+/[^/]+)(\.git)?$#https://github.com/\2#')
```

## 輸出變數

| 變數 | 說明 |
|---|---|
| `DEFAULT_BRANCH` | 遠端預設分支名（`main`／`master`）；無 remote 時為空 |
| `LATEST_TAG` | 最新 `vX.Y.Z` 標籤（有 remote 時限定 merged 進該分支）；無標籤則為 `v0.0.0` |
| `REMOTE_URL` | `https://github.com/{owner}/{repo}`；非 GitHub remote 時為原始 URL |
