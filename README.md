# blockchain

This repo documents my journey on learning how to create a solid blockchain using GO. The README will be written as I go along, and the information may change during its writting.

# Intro

## Blockchain

The concept of a blockchain is actually really simple. Just link together some "blocks" and you have yourself a block-`chain`. You can define a block by a structure that contains some data and some information relevant to the chain as a whole.

### Defining the structure of a block

The block I've implemented contains the minimum requirements to create a block.

```go
type Block struct {
	Hash     []byte // a hash of the Data + PrevHash
	Data     []byte
	PrevHash []byte // linked list functionality (chain)
}
```

- `Hash`: houses a slice of `byte` representative of the blocks's "unique" identity — the hashing algorithm used was `sha256`, not necessarily the most secure
- `Data`: the slice of `byte` that contains the information that's being stored by the block
- `PrevHash`: the slice of `byte` that references the hash of the previous block, ensuring order and forming a linked list

### Defining the blockchain

We can now define a blockchain as a list of blocks.

```go
type BlockChain struct {
	blocks []*Block
}
```

_An array of pointers is used to ensure no copies exist._

## Visualizing the blockchain

Necessary methods like `addBlock()`, `createBlock()` and `createBlockChain()` help us create and expand the block chain, while `printBlockChain()` aids in visualizing what's going on.

Let's test the following code:

```go
func main() {
	blockChain := createBlockChain() // the blockchain is created with a genesis block (1st block)

	blockChain.addBlock([]byte("hello blockchain"))
	blockChain.addBlock([]byte("I'm the third block"))
	blockChain.addBlock([]byte("and I'm the forth block"))

	blockChain.printBlockChain()
}

```

This is the output we got:

```
> go run main.go
--------
Previous Hash:
Data: GENSIS_BLOCK
Current Hash: 0e822cd4aef4ec6318fd2fdc641a01aaca763507060239505047847a6b25bd6b
--------
--------
Previous Hash: 0e822cd4aef4ec6318fd2fdc641a01aaca763507060239505047847a6b25bd6b
Data: hello blockchain
Current Hash: f3747b11556866834b7b3e739670371a51233ea50862acfc90fcc1b3e38571f9
--------
--------
Previous Hash: f3747b11556866834b7b3e739670371a51233ea50862acfc90fcc1b3e38571f9
Data: I'm the third block
Current Hash: 3424b464982a9654b54c1b8e41f49dd9880f040c4cd83dc81f39681072371acb
--------
--------
Previous Hash: 3424b464982a9654b54c1b8e41f49dd9880f040c4cd83dc81f39681072371acb
Data: and I'm the forth block
Current Hash: 5d2047642951b0bf3f55965df158f915131b1d20be126163eb3363f37c3cef14
--------
```

We can see out blockchain taking shape. The data is being stored inside blocks that correctly reference the prvious block in the blockchain.

> !> [!NOTE]
> Since the blockchain's hash is deterministic — the same input yields the same output — the code can be ran multiple times and the outout will be the exact same.
> Normally, blockchains have multiple copied ditributed amongst multiple distributed nodes. **This is what makes blockchains so secure**
> A node's blockchain may be compromised, i.e., its data tikered with, and have a node yield a diferent hash. Since that chain will be different from the N other exact copies of that chain, the error is quickly caught.

Let's test this further. I'll change the data of the second block to `"Hello blockchain"` with a capital `H`.

```
> go run main.go
--------
Previous Hash:
Data: GENSIS_BLOCK
Current Hash: 0e822cd4aef4ec6318fd2fdc641a01aaca763507060239505047847a6b25bd6b
--------
--------
Previous Hash: 0e822cd4aef4ec6318fd2fdc641a01aaca763507060239505047847a6b25bd6b
Data: Hello blockchain
Current Hash: de1ba7ef3cc33cd6ecec2ca19e34f4723cf15984b59b3d997553f3ccd719338b
--------
--------
Previous Hash: de1ba7ef3cc33cd6ecec2ca19e34f4723cf15984b59b3d997553f3ccd719338b
Data: I'm the third block
Current Hash: ec50c8c9dcf5f4f6f57f14978ca518511e2c863465d318a31b403efb549b6841
--------
--------
Previous Hash: ec50c8c9dcf5f4f6f57f14978ca518511e2c863465d318a31b403efb549b6841
Data: and I'm the forth block
Current Hash: 9628ec1c854a91f76e6fee6e37eaa220d7cf84834b3b66de77d30735fd7d095f
--------
```

If we now compare the hashes from this chain and the previous one, we see they differ starting from the second block.
