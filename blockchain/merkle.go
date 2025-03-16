package blockchain

import "crypto/sha256"

type MerkleTree struct {
	RootNode *MerkleNode
}

type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	newNode := MerkleNode{}

	// unpopulated children
	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		newNode.Data = hash[:]
	} else { // data corresponds to the concatenation of the children's hashes
		previousHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(previousHashes)
		newNode.Data = hash[:]
	}

	newNode.Left = left
	newNode.Right = right

	return &newNode
}

func newMerkleTree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode

	// check if the tree will not be balanced, i.e., the uneven
	// if so, duplicate the last entry
	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	// populate the tips of the tree
	for _, entry := range data {
		nodes = append(nodes, *NewMerkleNode(nil, nil, entry))
	}

	for range len(data) / 2 {
		var level []MerkleNode

		for j := 0; j < len(nodes); j += 2 {
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			level = append(level, *node)
		}

		nodes = level
	}

	tree := MerkleTree{&nodes[0]}

	return &tree
}
