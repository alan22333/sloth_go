# Sloth VDF in Go

这是一个基于论文 [A random zoo: sloth, uncorn, and trx](https://eprint.iacr.org/2015/366) 中 "Sloth" 算法的可验证延迟函数 (VDF) 的 Go 语言实现。

## 简介

可验证延迟函数 (VDF) 是一种特殊的密码学函数，其核心特性是：

- **计算慢**: 计算过程需要一段可预测的、不可并行的延迟时间。
- **验证快**: 一旦计算完成，任何人都可以快速地验证结果的正确性。

这种特性使其在需要公共可信随机数、共识协议以及防止恶意攻击的场景中非常有用。本项目提供了一个简单、健壮的 Go 库来实现此功能。

## 安装

```bash
go get -u github.com/alan22333/sloth_go
```


## 使用方法

使用本库通常分为三步：**创建实例**、**计算**和**验证**。

### 1. 创建 VDF 实例

首先，你需要一个符合 `p ≡ 3 (mod 4)` 条件的大素数和指定的延迟迭代次数来创建一个 `Sloth` 实例。

- **`GenerateSlothPrime(bits int)`**: 使用此辅助函数生成一个指定位数的、符合条件的密码学安全素数。
- **`New(p *big.Int, iterations int64)`**: 使用生成的素数 `p` 和迭代次数 `iterations` 来创建一个 `Sloth` VDF 实例。

### 2. 计算证明（慢）

使用 `Compute` 方法来执行耗时的 VDF 计算。

- **`Compute(input []byte)`**: 接收一个任意字节数组作为输入。
    - **返回**: 最终的 `hash`、用于验证的 `witness` 以及可能出现的 `error`。

### 3. 验证证明（快）

使用 `Verify` 方法来快速验证一个计算结果是否正确。

- **`Verify(input []byte, hash []byte, witness *big.Int)`**: 接收原始输入、计算得到的 `hash` 和 `witness`。
    - **返回**: 一个表示验证是否成功的 `bool` 值和可能出现的 `error`。

## API 概览

- `GenerateSlothPrime(bits int) (*big.Int, error)`: 生成 Sloth 专用的大素数。
- `New(p *big.Int, iterations int64) (*Sloth, error)`: 创建 VDF 实例。
- `(s *Sloth) Compute(input []byte) (hash []byte, witness *big.Int, err error)`: 执行耗时的计算。
- `(s *Sloth) Verify(input []byte, hash []byte, witness *big.Int) (bool, error)`: 执行快速验证。

## 测试

运行内置的测试来确保库的正确性和性能：

```bash
# 运行单元测试
go test -v

# 运行性能基准测试
go test -bench=.
```

## 许可证

本项目采用 [MIT License](LICENSE)。