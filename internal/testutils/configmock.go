package testutils

import "math/rand"

type MockConfigurationA struct {
	Value string
}

type MockConfigurationA2 MockConfigurationA
type MockConfigurationA3 MockConfigurationA
type MockConfigurationA4 MockConfigurationA

type MockConfigurationB struct {
	Value bool
}

type MockConfigurationWithTypedef struct {
	A2 MockConfigurationA2
	A3 MockConfigurationA3
	A4 MockConfigurationA4
}

type MockConfigurationWithOneDepthLevel struct {
	A MockConfigurationA
	B MockConfigurationB
}

type MockConfigurationWithA struct {
	A MockConfigurationA
}

type MockConfigurationWithPointer struct {
	PtrWithA  *MockConfigurationWithA
	PtrWithA2 *MockConfigurationWithA
}

type MockConfigurationWithDuplicates struct {
	A  MockConfigurationA
	A2 MockConfigurationA
	B  MockConfigurationB
}

type MockConfigurationWithTwoDepthLevels struct {
	First  MockConfigurationWithOneDepthLevel
	Second MockConfigurationWithOneDepthLevel
}

func randomString() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, 5)
	for i := range result {
		result[i] = letters[rand.Intn(len(letters))]
	}
	return string(result)
}

func randomBool() bool {
	return rand.Intn(2) == 0
}

func RandomMockConfigurationWithDuplicates() MockConfigurationWithDuplicates {
	return MockConfigurationWithDuplicates{
		A:  MockConfigurationA{Value: randomString()},
		A2: MockConfigurationA{Value: randomString()},
		B:  MockConfigurationB{Value: randomBool()},
	}
}

func RandomMockConfigurationWithTypedef() MockConfigurationWithTypedef {
	return MockConfigurationWithTypedef{
		A2: MockConfigurationA2(MockConfigurationA{Value: randomString()}),
		A3: MockConfigurationA3{Value: randomString()},
		A4: MockConfigurationA4(MockConfigurationA2{Value: randomString()}),
	}
}

func RandomMockConfigurationWithOneDepthLevel() MockConfigurationWithOneDepthLevel {
	return MockConfigurationWithOneDepthLevel{
		A: MockConfigurationA{Value: randomString()},
		B: MockConfigurationB{Value: randomBool()},
	}
}

func RandomMockConfigurationWithA() MockConfigurationWithA {
	return MockConfigurationWithA{
		A: MockConfigurationA{
			Value: randomString(),
		},
	}
}

func RandomMockConfigurationWithPointer() MockConfigurationWithPointer {
	a1 := RandomMockConfigurationWithA()
	a2 := RandomMockConfigurationWithA()
	return MockConfigurationWithPointer{
		PtrWithA:  &a1,
		PtrWithA2: &a2,
	}
}

func RandomMockConfigurationWithTwoDepthLevels() MockConfigurationWithTwoDepthLevels {
	return MockConfigurationWithTwoDepthLevels{
		First:  RandomMockConfigurationWithOneDepthLevel(),
		Second: RandomMockConfigurationWithOneDepthLevel(),
	}
}
