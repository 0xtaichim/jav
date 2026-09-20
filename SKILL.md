# JavCLI Agent Skill Definition

此文档定义了 AI Agent 如何使用 `javcli` 工具与 JavDB 数据库进行交互。

## 1. 技能描述 (Description)

**名称**: Jav Cli
**工具**: `javcli`
**功能**: 允许 Agent 搜索成人影片数据库、获取影片详细元数据（含磁力链接）、查看各类排行榜，以及管理收藏（包括想看，已看，演员）。
**输出格式**: JSON

## 2. 安装与环境准备 (Installation & Setup)

Agent 在使用前应检查工具是否可用。如果遇到 `command not found: javcli` 错误，请按以下步骤安装。

### 2.1 自动安装 (Auto Install)
前提：环境中必须安装了 Go (Golang 1.18+)。

```bash
# 方法 1: 直接通过 Go 安装 (推荐)
go install github.com/taichi/javcli@latest

# 方法 2: 源码编译 (如果方法 1 失败)
git clone https://github.com/taichi/javcli.git
cd javcli
go mod download
go build -o javcli
# 注意：如果使用源码编译，后续命令需使用 ./javcli 或将其移动到 PATH 路径下
```

### 2.2 验证安装
安装完成后，运行以下命令验证：
```bash
javcli --help
```

## 3. 核心能力与命令 (Capabilities & Commands)

Agent 应根据用户意图选择以下命令之一执行。所有命令均输出 JSON 数据，Agent 需解析 JSON 并以自然语言回答用户。

### 2.1 搜索影片 (Search)
用于查找特定番号或基于关键词的影片。

*   **命令**: `javcli search "<关键词或番号>"`
*   **示例**: `javcli search "SSNI-678"` 或 `javcli search "秘书"`
*   **返回关键字段**: `movies` (列表), `code`, `title`, `date`, `rating`

### 2.2 获取详情与磁力链 (Get Details)
用于获取特定番号的详细信息，**特别是磁力链接**和演员表。

*   **命令**: `javcli detail <番号>`
*   **示例**: `javcli detail SSNI-678`
*   **返回关键字段**:
    *   `title`, `code`, `date`, `duration`, `director`, `publisher`
    *   `actors`: 演员列表
    *   `magnets`: 磁力链接列表（包含 `magnet` 字符串, `size`, `is_hd` (高清), `has_subs` (字幕)）

### 2.3 查看排行榜 (View Rankings)
用于发现热门影片。

*   **命令**: `javcli rankings [flags]`
*   **参数**:
    *   `--period, -p`: 周期 (`daily`, `weekly`, `monthly`)。默认 `daily`。
    *   `--type, -t`: 类型 (`censored` [有码], `uncensored` [无码], `western` [欧美], `fc2`)。默认 `censored`。
*   **示例**:
    *   有码日榜: `javcli rankings`
    *   无码周榜: `javcli rankings -p weekly -t uncensored`
    *   FC2月榜: `javcli rankings -p monthly -t fc2`

## 3. 环境变量要求 (Environment)

确保运行环境已配置以下变量（通常在 Shell 会话中预设）：

```bash
# export SOCKS5_PROXY="127.0.0.1:6153"  # 可选, 代理访问
# export JAVDB_COOKIES=""          # 可选，用于管理收藏，获取无码/FC2数据
```

## 4. Agent 交互指南 (System Prompt)

将以下内容添加到 Agent 的系统提示词中：

```text
You have access to a CLI tool called 'javcli' for querying the JavDB database.
All 'javcli' commands output JSON. You must parse this JSON to answer user questions.

- To search: run `javcli search <query>`
- To get details (metadata, magnets, actors): run `javcli detail <code>`
- To view rankings: run `javcli rankings -p <period> -t <type>`
  - periods: daily, weekly, monthly
  - types: censored, uncensored, western, fc2

When providing magnet links from `javcli detail`:
- Prioritize links with `is_hd: true` (High Definition) and `has_subs: true` (Subtitles).
- Always display the file size.
```
