package main

import (
	"math/big"
	"testing"
)

// 全局变量，用于在测试和基准测试之间共享一个昂贵的 VDF 实例
var testVDF *Sloth
var testInput = []byte("A random zoo: sloth, unicorn, and trx")
var testPrimeBits = 64 // 使用较小的素数以加快测试速度
var testIterations int64 = 1000

func init() {
	// 在所有测试开始前，初始化一个 VDF 实例
	// 这避免了在每个测试和基准测试中都重新生成素数
	prime, err := GenerateSlothPrime(testPrimeBits)
	if err != nil {
		panic("Failed to generate prime for testing: " + err.Error())
	}
	testVDF, err = New(prime, testIterations)
	if err != nil {
		panic("Failed to create VDF for testing: " + err.Error())
	}
}

// TestComputeAndVerify_Correctness 检查一个正常的流程是否能通过
func TestComputeAndVerify_Correctness(t *testing.T) {
	t.Logf("Testing with %d-bit prime and %d iterations", testPrimeBits, testIterations)

	hash, witness, err := testVDF.Compute(testInput)
	if err != nil {
		t.Fatalf("Compute failed unexpectedly: %v", err)
	}

	verified, err := testVDF.Verify(testInput, hash, witness)
	if err != nil {
		t.Fatalf("Verify failed unexpectedly: %v", err)
	}

	if !verified {
		t.Error("Verification returned false, but expected true")
	}
	t.Log("Correctness test passed!")
}

// TestVerify_FailureCases 测试各种失败场景
func TestVerify_FailureCases(t *testing.T) {
	// 先生成一个有效的证明
	validHash, validWitness, err := testVDF.Compute(testInput)
	if err != nil {
		t.Fatalf("Setup for failure tests failed: %v", err)
	}

	// 定义测试用例
	testCases := []struct {
		name        string
		input       []byte
		hash        []byte
		witness     *big.Int
		expectError bool
	}{
		{
			name:        "Wrong Input",
			input:       []byte("wrong input data"),
			hash:        validHash,
			witness:     validWitness,
			expectError: true, // 预期会失败并返回错误
		},
		{
			name:        "Wrong Witness",
			input:       testInput,
			hash:        validHash,
			witness:     new(big.Int).Add(validWitness, big.NewInt(1)), // 修改 witness
			expectError: true,
		},
		{
			name:        "Wrong Hash",
			input:       testInput,
			hash:        []byte("wrong hash data"),
			witness:     validWitness,
			expectError: true,
		},
		{
			name:        "Nil Witness",
			input:       testInput,
			hash:        validHash,
			witness:     nil, // 模拟无效的 witness
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			verified, err := testVDF.Verify(tc.input, tc.hash, tc.witness)

			if verified {
				t.Error("Verification succeeded, but expected failure")
			}

			if tc.expectError && err == nil {
				t.Error("Expected an error, but got nil")
			}
		})
	}
	t.Log("Failure cases test passed!")
}

// TestNew_ParameterValidation 测试 New 函数的参数校验
func TestNew_ParameterValidation(t *testing.T) {
	// 1. 测试 p 不满足 p ≡ 3 (mod 4) 的情况
	// 找到一个 p ≡ 1 (mod 4) 的素数，例如 5, 13, 17...
	pNotCongruent, _ := new(big.Int).SetString("11", 16) // 17 = 4*4 + 1
	_, err := New(pNotCongruent, 100)
	if err == nil {
		t.Error("Expected error for p not congruent to 3 (mod 4), but got nil")
	}

	// 2. 测试 p 不是素数的情况
	pNotPrime, _ := new(big.Int).SetString("9", 10) // 9 is not prime
	_, err = New(pNotPrime, 100)
	if err == nil {
		t.Error("Expected error for non-prime p, but got nil")
	}

	// 3. 测试迭代次数为非正数
	prime, _ := GenerateSlothPrime(32)
	_, err = New(prime, 0)
	if err == nil {
		t.Error("Expected error for non-positive iterations, but got nil")
	}

	t.Log("Parameter validation test passed!")
}

// --- 基准测试 ---

// BenchmarkCompute 测试计算函数的性能
func BenchmarkCompute(b *testing.B) {
	// b.N 是由测试框架决定的迭代次数
	for i := 0; i < b.N; i++ {
		// 在基准测试中，我们不关心结果，只关心执行时间
		_, _, _ = testVDF.Compute(testInput)
	}
}

// BenchmarkVerify 测试验证函数的性能
func BenchmarkVerify(b *testing.B) {
	// 先计算一次，获取有效的 hash 和 witness
	hash, witness, err := testVDF.Compute(testInput)
	if err != nil {
		b.Fatalf("Setup for benchmark failed: %v", err)
	}

	// 重置计时器，因为我们不想把上面的 Compute 时间算进去
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = testVDF.Verify(testInput, hash, witness)
	}
}
