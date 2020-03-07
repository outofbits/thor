package utils

import "math/big"

func ex(intValMap map[string]interface{}, id func(val interface{}) string, ord func(a, b interface{}) bool) (interface{}, []string) {
    if len(intValMap) == 0 {
        return nil, []string{}
    }
    valMap := make(map[string][]string)
    var exVal interface{} = nil
    for key, value := range intValMap {
        if exVal == nil {
            exVal = value
        } else if ord(value, exVal) {
            list, found := valMap[id(value)]
            if found {
                valMap[id(value)] = append(list, key)
            } else {
                valMap[id(value)] = []string{key}
            }
            exVal = value
        }
    }
    keys, _ := valMap[id(exVal)]
    return exVal, keys
}

func cpInt(intValMap map[string]*big.Int) map[string]interface{} {
    m := make(map[string]interface{})
    for key, val := range intValMap {
        m[key] = val
    }
    return m
}

func cpFloat(intValMap map[string]*big.Float) map[string]interface{} {
    m := make(map[string]interface{})
    for key, val := range intValMap {
        m[key] = val
    }
    return m
}

func idInt(v interface{}) string {
    return v.(*big.Int).String()
}

func idFloat(v interface{}) string {
    return v.(*big.Float).String()
}

// scans the map with the corresponding int value, and
// then returns the maximum int value in the map as well as a
// list map keys that have the maximum int value.
func MaxInt(intValMap map[string]*big.Int) (*big.Int, []string) {
    ord := func(a, b interface{}) bool {
        return a.(*big.Int).Cmp(b.(*big.Int)) >= 0
    }
    val, list := ex(cpInt(intValMap), idInt, ord)
    if val != nil {
        return val.(*big.Int), list
    } else {
        return nil, list
    }
}

// scans the map with the corresponding int value, and
// then returns the minimum int value in the map as well as a
// list map keys that have the minimum int value.
func MinInt(intValMap map[string]*big.Int) (*big.Int, []string) {
    ord := func(a, b interface{}) bool {
        return a.(*big.Int).Cmp(b.(*big.Int)) <= 0
    }
    val, list := ex(cpInt(intValMap), idInt, ord)
    if val != nil {
        return val.(*big.Int), list
    } else {
        return nil, list
    }
}

// scans the map with the corresponding float value, and
// then returns the maximum float value in the map as well as a
// list map keys that have the maximum float value.
func MaxFloat(floatMap map[string]*big.Float) (*big.Float, []string) {
    ord := func(a, b interface{}) bool {
        return a.(*big.Float).Cmp(b.(*big.Float)) >= 0
    }
    val, list := ex(cpFloat(floatMap), idFloat, ord)
    if val != nil {
        return val.(*big.Float), list
    } else {
        return nil, list
    }
}

// scans the map with the corresponding float value, and
// then returns the minimum float value in the map as well as a
// list map keys that have the minimum float value.
func MinFloat(floatMap map[string]*big.Float) (*big.Float, []string) {
    ord := func(a, b interface{}) bool {
        return a.(*big.Float).Cmp(b.(*big.Float)) <= 0
    }
    val, list := ex(cpFloat(floatMap), idFloat, ord)
    if val != nil {
        return val.(*big.Float), list
    } else {
        return nil, list
    }
}

func MinMaxNormFloat(floatMap map[string]*big.Float) map[string]*big.Float {
    if len(floatMap) == 0 {
        return map[string]*big.Float{}
    }
    maxVal, _ := MaxFloat(floatMap)
    minVal, _ := MinFloat(floatMap)
    zero := maxVal.Cmp(new(big.Float).SetUint64(0)) == 0 && minVal.Cmp(new(big.Float).SetUint64(0)) == 0
    onlyMaxZero := !zero && maxVal.Cmp(new(big.Float).SetUint64(0)) == 0
    normMap := make(map[string]*big.Float)
    for key, value := range floatMap {
        if zero {
            normMap[key] = minVal
        } else if onlyMaxZero {
            normMap[key] = new(big.Float).Sub(new(big.Float).SetInt64(1),
                new(big.Float).Quo(new(big.Float).Sub(value, maxVal), minVal))
        } else {
            normMap[key] = new(big.Float).Quo(new(big.Float).Sub(value, minVal), maxVal)
        }
    }
    return normMap
}
