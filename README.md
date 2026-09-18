# kctx

`kctx` 用于扫描本机 kubeconfig 文件、查询 context，并展示每个 context 对应的自定义别名。

```text
CONTEXT        FILEPATH                           ENVIRONMENT
-------------  ---------------------------------  -----------
dev-cluster    /Users/you/.kube/dev-cluster.yaml  development
prod-cluster   /Users/you/.kube/prod-cluster.yaml  production
```

别名列是通用的：它可以叫 `ENVIRONMENT`、`REGION`、`OWNER` 或其他名称。

## 功能

- 扫描默认目录下所有 kubeconfig 文件。
- 不带参数时列出全部 context。
- 根据 context 名称进行大小写无关的模糊查询。
- 显示 context、kubeconfig 完整路径和自定义别名。
- 使用 `kctx config` 交互设置默认目录、别名列名和每个 context 的别名值。
- 支持 JSON 输出、自定义配置文件和临时覆盖 kubeconfig 目录。
- 不输出 token、证书或其他 kubeconfig 认证信息。
- 支持 Windows、macOS 和 Linux。

## 安装要求

- Go 1.23 或更高版本。

编译后的 `kctx` 不依赖 `kubectl`。

## 安装

```bash
go install github.com/WRAllen/kctx/cmd/kctx@latest
```

升级时再次执行同一条命令即可。

如果本机配置的第三方 Go 代理尚未同步这个版本并返回 404，可临时使用官方代理：

```bash
GOPROXY=https://proxy.golang.org,direct go install github.com/WRAllen/kctx/cmd/kctx@latest
```

### macOS 和 Linux

确保 Go 的二进制目录位于 `PATH`：

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

### Windows PowerShell

```powershell
$env:Path += ";$(go env GOPATH)\bin"
```

为了在新终端继续使用，请把 `$(go env GOPATH)\bin` 添加到 Windows 用户 `PATH`。

## 首次配置

运行交互式向导：

```bash
kctx config
```

交互示例：

```text
kctx configuration
Press Enter to keep the current value.

Kubeconfig directory [/Users/you/.kube]:
Alias column name [ENVIRONMENT]:
Found 24 contexts.
Configure alias values [Y/n]: y
Press Enter to keep a value; enter - to clear it.
ENVIRONMENT for dev-cluster [development]:
ENVIRONMENT for prod-cluster [production]:

Saved configuration to /Users/you/.config/kctx/config.yaml
```

操作规则：

- 直接按回车保留当前值。
- 输入新内容会覆盖当前值。
- 为某个 context 输入 `-` 会清除它的别名。
- 选择不配置别名时，只保存 kubeconfig 目录和别名列名。

默认配置文件位于用户目录：

```text
~/.config/kctx/config.yaml
```

Windows 上对应类似：

```text
C:\Users\your-name\.config\kctx\config.yaml
```

配置格式：

```yaml
kubeconfig_dir: /Users/you/.kube
alias:
  name: ENVIRONMENT
  values:
    dev-cluster: development
    prod-cluster: production
```

## 使用

列出全部 context：

```bash
kctx
```

模糊查询：

```bash
kctx prod
kctx readonly
kctx dev
```

临时扫描另一个目录，不改变保存的默认设置：

```bash
kctx --kube-dir /path/to/kubeconfigs
```

使用另一个配置文件：

```bash
kctx --config /path/to/config.yaml
kctx config --config /path/to/config.yaml
```

输出 JSON：

```bash
kctx --json minimax
```

JSON 中别名使用通用结构：

```json
[
  {
    "context": "prod-cluster",
    "filepath": "/Users/you/.kube/prod-cluster.yaml",
    "alias": {
      "name": "ENVIRONMENT",
      "value": "production"
    }
  }
]
```

查看版本和帮助：

```bash
kctx --version
kctx --help
kctx config --help
```
