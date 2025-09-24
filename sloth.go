package slothgo

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"math/big"
)

// Sloth 结构体持有 VDF 的所有参数
type Sloth struct {
	P          *big.Int // 大素数模数, p ≡ 3 (mod 4)
	Iterations int64    // 迭代次数 (延迟参数)
	HashFunc   func() hash.Hash

	// 预计算的值，用于加速
	sqrtExp *big.Int // (p+1)/4 用于计算平方根
}

// big.Int 常量
var (
	bigZero  = big.NewInt(0)
	bigOne   = big.NewInt(1)
	bigTwo   = big.NewInt(2)
	bigFour  = big.NewInt(4)
	bigThree = big.NewInt(3)
)

// New 创建一个新的 Sloth VDF 实例
// p: 十六进制表示的大素数
// iterations: 延迟循环的次数
func New(p *big.Int, iterations int64) (*Sloth, error) {
	if iterations <= 0 {
		return nil, errors.New("iterations must be positive")
	}

	// 验证 p 是一个素数且 p ≡ 3 (mod 4)
	if !p.ProbablyPrime(20) {
		return nil, errors.New("p is not a prime number")
	}
	if new(big.Int).Mod(p, bigFour).Cmp(bigThree) != 0 {
		return nil, errors.New("p must be congruent to 3 (mod 4)")
	}

	// 预计算 (p+1)/4 用于平方根
	sqrtExp := new(big.Int).Add(p, bigOne)
	sqrtExp.Div(sqrtExp, bigFour)

	return &Sloth{
		P:          p,
		Iterations: iterations,
		HashFunc:   sha256.New,
		sqrtExp:    sqrtExp,
	}, nil
}

// Compute (编码) 执行可验证延迟函数
// input: 任意字节数组作为输入
// 返回:
//   - hash: 最终输出的哈希值 (论文中的 g)
//   - witness: 用于验证的最终值 (论文中的 w)
//   - error: 计算过程中的错误
func (s *Sloth) Compute(input []byte) (hash []byte, witness *big.Int, err error) {
	// 步骤 1 & 3: h(s) 并转换为 w₀
	hasher := s.HashFunc()
	hasher.Write(input)
	uBytes := hasher.Sum(nil)

	w := new(big.Int).SetBytes(uBytes)
	w.Mod(w, s.P) // w₀ = int(h(s))

	// 步骤 4: 迭代 l 次
	for i := int64(0); i < s.Iterations; i++ {
		w = s.tau(w)
	}

	witness = new(big.Int).Set(w)

	// 步骤 5: 计算最终哈希 g = h(hex(wₗ))
	hasher.Reset()
	hasher.Write(witness.Bytes())
	hash = hasher.Sum(nil)

	return hash, witness, nil
}

// Verify (解码/验证) 验证 VDF 的输出是否正确
// input: 原始输入
// hash: Compute 函数返回的哈希值
// witness: Compute 函数返回的见证
// 返回:
//   - bool: 验证是否成功
//   - error: 验证过程中的错误
func (s *Sloth) Verify(input []byte, hash []byte, witness *big.Int) (bool, error) {
	if input == nil {
		return false, errors.New("input cannot be nil")
	}
	if hash == nil {
		return false, errors.New("hash cannot be nil")
	}
	if witness == nil {
		return false, errors.New("witness cannot be nil")
	}
	// 确保 witness 在 F_p 域内，这是一个很好的健壮性检查
	if witness.Cmp(s.P) >= 0 || witness.Sign() < 0 {
		return false, fmt.Errorf("witness must be in the range [0, p-1]")
	}

	// 验证 g = h(hex(w))
	hasher := s.HashFunc()
	hasher.Write(witness.Bytes())
	expectedHash := hasher.Sum(nil)
	if !bytes.Equal(hash, expectedHash) {
		return false, errors.New("hash of witness does not match provided hash")
	}

	// 步骤 4 & 5 (逆向): 从 w 开始，迭代 l 次 τ⁻¹
	wCheck := new(big.Int).Set(witness)
	for i := int64(0); i < s.Iterations; i++ {
		wCheck = s.tauInverse(wCheck)
	}

	// 计算预期的初始值 w₀
	hasher.Reset()
	hasher.Write(input)
	uBytes := hasher.Sum(nil)
	wStartExpected := new(big.Int).SetBytes(uBytes)
	wStartExpected.Mod(wStartExpected, s.P)

	// 比较逆向计算的结果和预期的初始值
	if wCheck.Cmp(wStartExpected) == 0 {
		return true, nil
	}

	return false, errors.New("verification failed: reversed witness does not match initial value")
}

// sigma (σ) 实现 "邻居交换" 置换
// 如果 x_hat 是偶数, σ(x) = x - 1
// 如果 x_hat 是奇数, σ(x) = x + 1
func (s *Sloth) sigma(x *big.Int) *big.Int {
	// big.Int.Bit(0) 返回最低有效位, 0 表示偶数, 1 表示奇数
	res := new(big.Int)
	if x.Bit(0) == 0 { // 偶数
		res.Sub(x, bigOne)
	} else { // 奇数
		res.Add(x, bigOne)
	}
	// 确保结果在 F_p 域内
	return res.Mod(res, s.P)
}

// sigmaInverse (σ⁻¹) "邻居交换"的逆也是它本身
func (s *Sloth) sigmaInverse(x *big.Int) *big.Int {
	return s.sigma(x)
}

// rho (ρ) 计算具有偶数提升值的模平方根
func (s *Sloth) rho(x *big.Int) *big.Int {
	// 检查 x 是否是二次剩余
	valToRoot := new(big.Int)
	if big.Jacobi(x, s.P) == 1 {
		valToRoot.Set(x)
	} else {
		// 如果不是，取 -x 的根
		valToRoot.Neg(x).Mod(valToRoot, s.P)
	}

	// 计算根 y = valToRoot^((p+1)/4) mod p
	root := new(big.Int).Exp(valToRoot, s.sqrtExp, s.P)

	// 选择偶数提升值的根
	if root.Bit(0) == 0 { // 偶数
		return root
	}
	// 否则，另一个根是 p - root，它一定是偶数
	return new(big.Int).Sub(s.P, root)
}

// rhoInverse (ρ⁻¹) 是 ρ 的逆运算
// 如果 y_hat 是偶数, ρ⁻¹(y) = y²
// 如果 y_hat 是奇数, ρ⁻¹(y) = -y²
func (s *Sloth) rhoInverse(y *big.Int) *big.Int {
	ySquared := new(big.Int).Exp(y, bigTwo, s.P)
	if y.Bit(0) == 0 { // 偶数
		return ySquared
	}
	// 奇数
	return new(big.Int).Neg(ySquared).Mod(ySquared, s.P)
}

// tau (τ) 是核心的迭代函数
func (s *Sloth) Tau(x *big.Int) *big.Int {
	return s.rho(s.sigma(x))
}

// tauInverse (τ⁻¹) 是 τ 的逆函数
func (s *Sloth) TauInverse(y *big.Int) *big.Int {
	return s.sigmaInverse(s.rhoInverse(y))
}

// GenerateSlothPrime 生成一个满足 p ≡ 3 (mod 4) 的大素数
func GenerateSlothPrime(bits int) (*big.Int, error) {
	for {
		// 1. 生成一个随机素数
		prime, err := rand.Prime(rand.Reader, bits)
		if err != nil {
			return nil, fmt.Errorf("failed to generate candidate prime: %w", err)
		}

		// 2. 检查条件 p ≡ 3 (mod 4)
		//    new(big.Int).Mod(prime, bigFour) 计算 prime % 4
		//    .Cmp(bigThree) == 0 检查结果是否等于 3
		if new(big.Int).Mod(prime, bigFour).Cmp(bigThree) == 0 {
			// 3. 找到符合条件的素数，返回
			return prime, nil
		}

		// 如果不符合，循环继续
	}
}
