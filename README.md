This repo documents my journey on learning how to create a solid blockchain using GO. The README will be written as I go along, and the information may change during its writing.

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

Necessary methods like `addBlock()`, `createBlock()` and `createBlockChain()` help us create and expand the blockchain, while `printBlockChain()` aids in visualizing what's going on.

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

We can see our blockchain taking shape. The data is being stored inside blocks that correctly reference the previous block in the blockchain.

Since the blockchain's hash is deterministic — the same input yields the same output — we can run the code multiple times and expect the same output. Normally, blockchains also have multiple copies distributed amongst various nodes.
**This is what makes blockchains so secure**.
A node's blockchain may be compromised and have a node yield a different hash, i.e., its data tinkered with. Since that chain will be different from the N other exact copies of that chain, the error is quickly caught.

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

If we compare the hashes from this chain and the previous one, we can see that they differ starting from the second block.

# Proof-of-work

In simple terms, proof-of-work ensure that a network's node has lended enough computational power (`work`) to create a block. This newly created block can quickly be validated and `proves` the work computational power was indeed expended.

## Why use proof-of-work?

The way its set up, to create a new block one must spend enough computational power to find a hash that meets certain criteria. The work that's put into this node is extremelly harduous.

## So, what's the actual "work"?

Before we go further, let's add the parameter `nonce` to our block structure.

```go
type Block struct {
	Hash     []byte // a hash of the Data + PrevHash
	Data     []byte
	PrevHash []byte // linked list functionality (chain)
    Nonce    int
}
```

This value represents a sort of iteration count of the block, and is a way to get different hash values from the same contents of the block.

We can also define a structure for a proof-of-work.

```go
type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}
```

- `Block`: the block that's trying to prove its work
- `Target`: an integer treshold which a block's hash must confine to

Let's take a closer look at what a target might look like. First, let's define a difficulty level from 0-256

```go
const Difficulty = 10
```

The value of the `Target` parameter is determined by the inverse of the difficulty.

```go
func NewProof(b *Block) *ProofOfWork {
	target := big.NewInt(1)

	// left shift the bytes 256-Difficulty times
	// 256 is used because it represents the size of the block's hash
	// the number 00000000....000001 would now be 0000...0001000...000000
	//            ^256th bit       ^1st bit                 ^"256-difficulty"th bit
	target.Lsh(target, uint(256-Difficulty))

	return &ProofOfWork{b, target}
}
```

A block now has the goal of creating a hash that, **in this project**, has a lower numerical value than the target. Let's break down what this means.

Say we have a node with data `"Hello World!"` and a starting `nonce` of `0`. Its hash will be computed like so:

```
hash(block.PrevHash, block.PrevHash, Nonce, Difficulty) --> hash(..., "Hello World!", 0, 12)
```

Now, let's admit the output was the following:

```
2be9e3e49cba8bcdc3c8a1e08d11fa520909249af0f82f409d0e412b83f0adb7
```

Since out `Difficulty` is set to `12`, it's hexadecimal value has only `62` digits instead of `64`.

```
10000000000000000000000000000000000000000000000000000000000000
```

We can now compare our block's hash's output to the target, and check if it's inferior. Since our output has 64 digits and no leading 0s, the block's hash did not meet the target.
This make the process repeat all over again, but now with a nonve of `1`. This gives a completely different hash that may be less than the target.
This whole mechanism is tucked away inside of the `Run()` method.

Let's see it in action. With `Difficulty` set to `12`, we get the following:
![proof-of-work showcase](assets/pow-shocase.mov))

Now, with `Difficulty` set to `20` — a marginal increase — our program takes a bit more time:
![proof-of-work showcase 2](assets/pow-shocase-2.mov))

## And why is proof-of-work so safe?

Imagine we now dial up our `Difficulty` up to `64`. This would result in a target number that only has `49` digits.

```
1000000000000000000000000000000000000000000000000
```

Comparing to the previous target, the number of acceptable values has gone from `2^4^62` to `2^4^49`.
Since our hashes have `256 bits`, the probability of finding a hash has decreased from about `25%` all the down to `0.003%`
The amount of time wasted grows exponentially, and the target´s difficulty increases with the lifespan of the blockchain.

### What if I was to attack the blockchain?

Let's say you wanted to re-write the past and change a double the amount of a transaction you did a while a go.

1. Starting with that block, you would have to re-hash every single block up-until the present. This would take an unprecedented amount of work.
2. Since PoW blockchains determine the vali chain as the longest one (i.e., the one with the most accumulated work), you would have to re-hash every block faster than every other miner combined. Since blockchains are mined by millions of distributed miners, you would need at leat `51%` of the network's computational power.
3. Even if you could manage to aquire said power, no financial gain could even be garantied.
4. And, worse of all, if an event like this happens, the network can ignore the your chain and switch to an honest fork
