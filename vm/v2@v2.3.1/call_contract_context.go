package vm

import (
	"fmt"

	"chainmaker.org/chainmaker/pb-go/v2/common"
)

const (
	one               uint64 = 1
	depthShift               = 60
	rtIndexShift             = 44
	rtIndexHighestPos        = 59
	runtimeTypeLen           = 4
	rtIndexMask              = one<<depthShift - one<<rtIndexShift
	rtListMask               = one<<rtIndexShift - 1
	latestLayerMask          = one<<runtimeTypeLen - 1
)

// CallContractContext maintains a bitmap of call contract info, the bitmap can easily
// and quickly update and query call contract context such as type and depth. exg:
// value:	0011-00100010-0000000000000000000000000000000000000000 0110 0010 0110
// meaning:	3times - use wasmer and dockergo - dockergo -> wasmer -> dockergo
type CallContractContext struct {
	// 63          59           43                   0
	// +----------+^-----------+-^---------+---------
	// |   4bits   |   16bits    |   	44bits     	|
	// +----------+^-----------+-^---------+---------
	//    depth   |   rt_index   |      rt_list     |
	//
	// depth record the length of vec<runtime_type>
	// rt_index record the virtual machine type before the current layer
	// rt_list excluding the runtime_type of the current layer
	// one runtimetype occupies 4 bits
	ctxBitmap uint64
}

// NewCallContractContext is used to create a new CallContractContext
func NewCallContractContext(ctxBitmap uint64) *CallContractContext {
	return &CallContractContext{ctxBitmap: ctxBitmap}
}

// AddLayer add new call contract layer to ctxBitmap
func (c *CallContractContext) AddLayer(runtimeType common.RuntimeType) {

	// head of new bitmap, new depth
	currDepth := c.GetDepth()
	depth := (currDepth + 1) << depthShift

	// rtIndex of new bitmap, new type bit map
	var rtIndex uint64
	if currDepth != 0 {
		//update current type and rt index first id depth > 0
		currentType := c.ctxBitmap & latestLayerMask
		rtIndex = c.ctxBitmap&rtIndexMask | (1 << (rtIndexHighestPos - currentType))
	}

	// rtList of new bit map, new type queue
	rtList := (c.ctxBitmap&rtListMask)<<runtimeTypeLen | uint64(runtimeType)

	c.ctxBitmap = depth | rtIndex | rtList
}

// ClearLatestLayer clear the latest call contract layer from bitmap
func (c *CallContractContext) ClearLatestLayer() {
	currDepth := c.GetDepth()
	depth := (currDepth - 1) << depthShift

	// TODO: optimize
	rtQueue := c.GetAllTypes()
	var rtIndex uint64
	newDepth := int(currDepth) - 2

	//for every depth, type bit and rt index need update
	for i := 0; i < newDepth; i++ {
		typeBit := uint64(1 << (rtIndexHighestPos - rtQueue[i]))
		rtIndex |= typeBit
	}

	rtList := c.ctxBitmap & rtListMask >> runtimeTypeLen
	c.ctxBitmap = depth | rtIndex | rtList
}

// GetDepth returns current depth
func (c *CallContractContext) GetDepth() uint64 {
	return c.ctxBitmap >> depthShift
}

// HasUsed judge whether a vm has been used
func (c *CallContractContext) HasUsed(runtimeType common.RuntimeType) bool {
	rtIndexBit := uint64(1 << (rtIndexHighestPos - runtimeType))
	return rtIndexBit&c.ctxBitmap > 0
}

// GetTypeByDepth get runtime type by depth
func (c *CallContractContext) GetTypeByDepth(depth uint64) (common.RuntimeType, error) {

	currDepth := c.GetDepth()

	//current depth can not < 0
	if depth > currDepth || depth == 0 {
		return 0, fmt.Errorf("depth overflow, current depth: %d, input depth: %d", currDepth, depth)
	}

	runtimeType := c.ctxBitmap >> ((currDepth - depth) * runtimeTypeLen) & latestLayerMask
	return common.RuntimeType(runtimeType), nil
}

// GetAllTypes return all types ordered by depth
func (c *CallContractContext) GetAllTypes() []common.RuntimeType {

	currDepth := c.GetDepth()
	types := make([]common.RuntimeType, currDepth)
	bitmap := c.ctxBitmap
	var i uint64

	//update types for every depth
	for ; i < currDepth; i++ {
		types[currDepth-i-1] = common.RuntimeType(bitmap & latestLayerMask)
		bitmap >>= runtimeTypeLen
	}
	return types
}

// GetCtxBitmap returns ctx bitmap
func (c *CallContractContext) GetCtxBitmap() uint64 {
	return c.ctxBitmap
}
