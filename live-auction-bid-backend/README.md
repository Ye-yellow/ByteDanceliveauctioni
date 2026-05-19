# 直播互动竞拍系统后端

当前阶段：需求先行。

本仓库将作为比赛项目后端仓库。代码实现暂缓，先在 `docs/openclaw/v1/` 下完成 V1 需求草案、范围确认和架构边界，再逐步落地。

## 当前文档

```text
docs/openclaw/v1/01-需求草案.md
```

## 协作规则

- 文档默认中文。
- OpenClaw 产出的分析文档放在 `docs/openclaw/` 下。
- 先确认需求，再写架构，再写代码。

## 本地运行

```bash
# 如果系统没有 go，可使用 OpenClaw 用户态 go：
export PATH=/home/ye/OpenClaw/state/tools/go/bin:$PATH

go run ./app/auction/service/cmd/server
```

默认监听：`http://127.0.0.1:18080`。

V1 当前是内存版闭环，重启后数据会清空。
