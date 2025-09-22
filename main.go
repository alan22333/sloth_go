package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	bits := 256 // Sloth VDF 通常使用 256 位或更大的素数
	fmt.Printf("Searching for a %d-bit prime p where p ≡ 3 (mod 4)...\n\n", bits)

	prime, err := GenerateSlothPrime(bits)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	p := prime
	iterations := 100000 // 10万次迭代用于演示，真实场景会大得多

	fmt.Println("VDF Sloth PoC")
	fmt.Printf("Prime (p): %s\n", p)
	fmt.Printf("Iterations (l): %d\n\n", iterations)

	// --- 初始化 ---
	slothVDF, err := New(p, int64(iterations))
	if err != nil {
		log.Fatalf("Failed to create Sloth VDF: %v", err)
	}

	input := []byte("hello verifiable delay functions")
	fmt.Printf("Input message: \"%s\"\n\n", string(input))

	// --- 计算 (编码) ---
	fmt.Println("Computing VDF... (this will take a moment)")
	startTime := time.Now()
	hash, witness, err := slothVDF.Compute(input)
	computeTime := time.Since(startTime)

	if err != nil {
		log.Fatalf("Compute failed: %v", err)
	}

	fmt.Printf("Compute successful in %s\n", computeTime)
	fmt.Printf("  - Final Hash (g): %x\n", hash)
	fmt.Printf("  - Witness    (w): %x\n\n", witness)

	// --- 验证 (解码) ---
	fmt.Println("Verifying VDF...")
	startTime = time.Now()
	verified, err := slothVDF.Verify(input, hash, witness)
	verifyTime := time.Since(startTime)

	if err != nil {
		log.Fatalf("Verify failed: %v", err)
	}

	fmt.Printf("Verification result: %t\n", verified)
	fmt.Printf("Verify successful in %s\n\n", verifyTime)

	fmt.Printf("Time comparison: Compute took %.2f times longer than Verify.\n", computeTime.Seconds()/verifyTime.Seconds())

	// --- 尝试一个错误的验证 ---
	fmt.Println("\n--- Testing verification with wrong input ---")
	wrongInput := []byte("this is not the correct input")
	verified, err = slothVDF.Verify(wrongInput, hash, witness)
	if err != nil {
		fmt.Printf("Verification with wrong input failed as expected: %v\n", err)
	} else if !verified {
		fmt.Println("Verification with wrong input correctly returned false.")
	}

}
