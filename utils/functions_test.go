package utils

import (
    "github.com/stretchr/testify/assert"
    "math/big"
    "testing"
)

func TestMaxInt_FullMap_mustReturnMaxValueAndCorrectKeys(t *testing.T) {
    input := map[string]*big.Int{
        "a": new(big.Int).SetUint64(2),
        "b": new(big.Int).SetUint64(6),
        "c": new(big.Int).SetUint64(1),
        "d": new(big.Int).SetUint64(3),
        "e": new(big.Int).SetUint64(1),
        "f": new(big.Int).SetUint64(6),
    }
    maxVal, keys := MaxInt(input)
    if assert.NotNil(t, maxVal) {
        assert.Equal(t, uint64(6), maxVal.Uint64())
    }
    if assert.NotNil(t, keys) {
        assert.ElementsMatch(t, []string{"b", "f"}, keys)
    }
}

func TestMaxInt_SingletonMap_mustReturnMaxValueAndCorrectKeys(t *testing.T) {
    input := map[string]*big.Int{
        "a": new(big.Int).SetUint64(2),
    }
    maxVal, keys := MaxInt(input)
    if assert.NotNil(t, maxVal) {
        assert.Equal(t, uint64(2), maxVal.Uint64())
    }
    if assert.NotNil(t, keys) {
        assert.ElementsMatch(t, []string{"a"}, keys)
    }
}

func TestMaxInt_EmptyMap_mustReturnNilAndEmptyList(t *testing.T) {
    maxVal, keys := MaxInt(map[string]*big.Int{})
    assert.Nil(t, maxVal)
    assert.Nil(t, keys)
}

func TestMinInt_FullMap_mustReturnMinValueAndCorrectKeys(t *testing.T) {
    input := map[string]*big.Int{
        "a": new(big.Int).SetUint64(2),
        "b": new(big.Int).SetUint64(4),
        "c": new(big.Int).SetUint64(1),
        "d": new(big.Int).SetUint64(6),
        "e": new(big.Int).SetUint64(1),
        "f": new(big.Int).SetUint64(5),
    }
    minVal, keys := MinInt(input)
    if assert.NotNil(t, minVal) {
        assert.Equal(t, uint64(1), minVal.Uint64())
    }
    if assert.NotNil(t, keys) {
        assert.ElementsMatch(t, []string{"c", "e"}, keys)
    }
}

func TestMinInt_SingletonMap_mustReturnMinValueAndCorrectKeys(t *testing.T) {
    input := map[string]*big.Int{
        "a": new(big.Int).SetUint64(2),
    }
    minVal, keys := MinInt(input)
    if assert.NotNil(t, minVal) {
        assert.Equal(t, uint64(2), minVal.Uint64())
    }
    if assert.NotNil(t, keys) {
        assert.ElementsMatch(t, []string{"a"}, keys)
    }
}

func TestMinInt_EmptyMap_mustReturnNilAndEmptyList(t *testing.T) {
    maxVal, keys := MinInt(map[string]*big.Int{})
    assert.Nil(t, maxVal)
    assert.Nil(t, keys)
}

func TestMaxFloat_FullMap_mustReturnMinValueAndCorrectKeys(t *testing.T) {
    input := map[string]*big.Float{
        "a": new(big.Float).SetFloat64(2.3333),
        "b": new(big.Float).SetFloat64(4.25),
        "c": new(big.Float).SetFloat64(4.25),
        "d": new(big.Float).SetFloat64(6.4),
        "e": new(big.Float).SetFloat64(2.3333),
        "f": new(big.Float).SetFloat64(5.0),
    }
    maxVal, keys := MaxFloat(input)
    if assert.NotNil(t, maxVal) {
        v, _ := maxVal.Float64()
        assert.Equal(t, 6.4, v)
    }
    if assert.NotNil(t, keys) {
        assert.ElementsMatch(t, []string{"d"}, keys)
    }
}

func TestMaxFloat_SingletonMap_mustReturnMinValueAndCorrectKeys(t *testing.T) {
    input := map[string]*big.Float{
        "d": new(big.Float).SetFloat64(0),
    }
    maxVal, keys := MaxFloat(input)
    if assert.NotNil(t, maxVal) {
        v, _ := maxVal.Float64()
        assert.Equal(t, 0.0, v)
    }
    if assert.NotNil(t, keys) {
        assert.ElementsMatch(t, []string{"d"}, keys)
    }
}

func TestMaxFloat_EmptyMap_mustReturnNilAndEmptyList(t *testing.T) {
    maxVal, keys := MinFloat(map[string]*big.Float{})
    assert.Nil(t, maxVal)
    assert.Nil(t, keys)
}

func TestMinFloat_FullMap_mustReturnMinValueAndCorrectKeys(t *testing.T) {
    input := map[string]*big.Float{
        "a": new(big.Float).SetFloat64(2.3333),
        "b": new(big.Float).SetFloat64(4.25),
        "c": new(big.Float).SetFloat64(4.25),
        "d": new(big.Float).SetFloat64(6.4),
        "e": new(big.Float).SetFloat64(2.3333),
        "f": new(big.Float).SetFloat64(5.0),
    }
    minVal, keys := MinFloat(input)
    if assert.NotNil(t, minVal) {
        v, _ := minVal.Float64()
        assert.Equal(t, 2.3333, v)
    }
    if assert.NotNil(t, keys) {
        assert.ElementsMatch(t, []string{"a", "e"}, keys)
    }
}

func TestMinFloat_EmptyMap_mustReturnNilAndEmptyList(t *testing.T) {
    minVal, keys := MinFloat(map[string]*big.Float{})
    assert.Nil(t, minVal)
    assert.Nil(t, keys)
}

func TestMinMaxFloat_FullMap_mustReturnNormMap(t *testing.T) {
    input := map[string]*big.Float{
        "a": new(big.Float).SetFloat64(0),
        "b": new(big.Float).SetFloat64(4.25),
        "c": new(big.Float).SetFloat64(12.2),
        "d": new(big.Float).SetFloat64(6.4),
        "e": new(big.Float).SetFloat64(2.3333),
        "f": new(big.Float).SetFloat64(24.4),
    }
    normInput := MinMaxNormFloat(input)
    if assert.NotNil(t, normInput) {
        v, found := normInput["c"]
        assert.True(t, found)
        if assert.NotNil(t, v) {
            fv, _ := v.Float64()
            assert.Equal(t, 0.5, fv)
        }
    }
}

func TestMinMaxFloat_AllZeroMap_mustReturnNormMapWithAllZero(t *testing.T) {
    input := map[string]*big.Float{
        "a": new(big.Float).SetFloat64(0.0),
        "b": new(big.Float).SetFloat64(0.0),
        "c": new(big.Float).SetFloat64(0.0),
        "d": new(big.Float).SetFloat64(0.0),
    }
    normInput := MinMaxNormFloat(input)
    if assert.NotNil(t, normInput) {
        v, found := normInput["c"]
        assert.True(t, found)
        if assert.NotNil(t, v) {
            fv, _ := v.Float64()
            assert.Equal(t, 0.0, fv)
        }
    }
}

func TestMinMaxFloat_ZeroMaxMap_mustReturnNormMap(t *testing.T) {
    input := map[string]*big.Float{
        "a": new(big.Float).SetFloat64(-12.2),
        "b": new(big.Float).SetFloat64(0.0),
        "c": new(big.Float).SetFloat64(-6.1),
        "d": new(big.Float).SetFloat64(-3.0),
    }
    normInput := MinMaxNormFloat(input)
    if assert.NotNil(t, normInput) {
        v, found := normInput["c"]
        assert.True(t, found)
        if assert.NotNil(t, v) {
            fv, _ := v.Float64()
            assert.Equal(t, 0.5, fv)
        }
    }
}

func TestMinMaxFloat_SingletonMaxMap_mustReturnNormMap(t *testing.T) {
    input := map[string]*big.Float{
        "a": new(big.Float).SetFloat64(4.0),
    }
    normInput := MinMaxNormFloat(input)
    if assert.NotNil(t, normInput) {
        v, found := normInput["a"]
        assert.True(t, found)
        if assert.NotNil(t, v) {
            fv, _ := v.Float64()
            assert.Equal(t, 0.0, fv)
        }
    }
}