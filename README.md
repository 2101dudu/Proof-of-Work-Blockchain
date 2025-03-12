This repo documents my journey in learning how to create a solid blockchain using GO. The README will be written as I go along, and the information may change during its writing.

# Intro

## Blockchain

The concept of a blockchain is actually really simple. Just link together some "blocks" and you have yourself a block-`chain`. You can define a block as a structure that contains some data and some information relevant to the chain as a whole.

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

Necessary methods like `addBlock()`, `createBlock()`, and `createBlockChain()` help us create and expand the blockchain, while `printBlockChain()` aids in visualizing what's going on.

Let's test the following code:

```go
func main() {
	blockChain := createBlockChain() // the blockchain is created with a genesis block (1st block)

	blockChain.addBlock([]byte("hello blockchain"))
	blockChain.addBlock([]byte("I'm the third block"))
	blockChain.addBlock([]byte("and I'm the fourth block"))

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
Data: and I'm the fourth block
Current Hash: 5d2047642951b0bf3f55965df158f915131b1d20be126163eb3363f37c3cef14
--------
```

We can see our blockchain taking shape. The data is stored inside blocks that correctly reference the previous block in the blockchain.

Since the blockchain's hash is deterministic — the same input yields the same output — we can run the code multiple times and expect the same output. Normally, blockchains also have multiple copies distributed amongst various nodes.
**This makes blockchains really secure**.
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
Data: and I'm the fourth block
Current Hash: 9628ec1c854a91f76e6fee6e37eaa220d7cf84834b3b66de77d30735fd7d095f
--------
```

If we compare the hashes from this chain and the previous one, we can see that they differ starting from the second block.

# Proof-of-work

In simple terms, proof-of-work ensures that a network's node has provided enough computational power (`work`) to create a block. This newly created block can quickly be validated and `proves` the work computational power was indeed expended.

## Why use proof-of-work?

The way it's set up, to create a new block one must spend enough computational power to find a hash that meets certain criteria. The work that's put into this node is extremely arduous.

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

This value represents a sort of iteration count of the block and is a way to get different hash values from the same contents of the block.

We can also define a structure for a proof-of-work.

```go
type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}
```

- `Block`: the block that's trying to prove its work
- `Target`: an integer threshold which a block's hash must confine to

Let's take a closer look at what a target might look like. First, let's define a difficulty level from 0-256

```go
const Difficulty = 12
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
hash(block.PrevHash, block.Data, Nonce, Difficulty) --> hash(..., "Hello World!", 0, 12)
```

Now, let's admit the output was the following:

```
2be9e3e49cba8bcdc3c8a1e08d11fa520909249af0f82f409d0e412b83f0adb7
```

Since our `Difficulty` is set to `12`, its hexadecimal value has `2` leading zeros, making it only `62` digits instead of `64`.

```
10000000000000000000000000000000000000000000000000000000000000
```

We can now compare our block's hash's output to the target, and check if it's inferior. Since our output has 64 digits and no leading 0s, the block's hash did not meet the target.
This makes the process start over again, but now with a nonce of `1`. This gives a completely different hash that may be less than the target.
This whole mechanism is tucked away inside of the `Run()` method.

Let's see it in action. With `Difficulty` set to `12`, we get the following:

https://github.com/user-attachments/assets/842cece7-ba54-4648-bb33-583bf88b440a

Now, with `Difficulty` set to `20` — a marginal increase — our program takes a bit more time:

https://github.com/user-attachments/assets/ebd7511f-e5b0-4820-a9cd-75494ffd64c1

## And why is proof-of-work so safe?

Imagine we now dial up our `Difficulty` up to `64`.

```
1000000000000000000000000000000000000000000000000
```

Compared to the previous target, the number of acceptable values has gone from `2^(256-12)` to `2^(256-20)`.
Since our hashes have `256 bits`, the probability of finding a hash has decreased from about `0.024%` down to `0.000095%`
The amount of time wasted grows exponentially, and the target´s difficulty increases with the lifespan of the blockchain.

### What if I was to attack the blockchain?

Let's say you wanted to re-write the past and double the amount of a transaction you did a while ago.

1. Starting with that block, you would have to re-hash every single block up until the present. This would take an unprecedented amount of work.
2. Since PoW blockchains determine the valid chain as the longest one (i.e., the one with the most accumulated work), you would have to re-hash every block faster than every other miner combined. Since blockchains are mined by millions of distributed miners, you would need at least `51%` of the network's computational power.
3. Even if you could manage to acquire said power, no financial gain could even be guaranteed.
4. And, worse of all, if an event like this happens, the network can ignore your chain and switch to an honest fork

# Storing the blockchain

A blockchain is nothing without a persistent memory storing it. To this end, we will use the native GO database library `BadgerDB` to handle the read-write storage of our blockchain.
BadgerDB works with `key: value` pairs and stores the bytes in regular files and folders. Since BadgerDB only works with slices of `byte`, we need to refactor some of our structs data handlings.

Using `encoding/gob`, we can now use the `Serialize()` and `Deserialize()` methods to translate data to and from byte structures.

We can also update our blockchain structures by including a parameter referencing the Badger database.

```go
const (
	dbPath = "./tmp/blocks"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}
```

This means that the blockchain no longer keeps in memory the totality of blocks, but rather reads and writes to the database when necessary.
We also need to keep track of which block the blockchain is at, so we added a parameter `LastHash`.

With this done, we need to rethink our blockchain methods. The `CreateBlockChain()` and `AddBlock()` methods now need to read and write from the DB as needed.

In the meantime, we lost the ability to iterate over our blockchains' blocks.
We can recoup this functionality by creating a helper structure,

```go
type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}
```

and make use of the `Iterator()` and `Next()` methods to iterate over every block.
This completes the persistent storage part, but now we can't really see what's going on.

# CLI

We can implement a simple CLI to remove hardcoded blockchain methods and give the program's users added functionalities.
Using `cli.Run()` on the `main()` method, our program now offers the users the ability to `print` the blockchain or `add` a new block with a given `data`. The full usage is as follows:

```
Usage:
    add -block BLOCK_DATA — adds a block to the blockchain
    print — Prints the blocks in the blockchain
```

Let's put it in action!

In this first example, the database has no entry, so a genesis node needs to be created and, most importantly, sign its proof-of-work.

https://github.com/user-attachments/assets/5f0d926a-a435-4a82-9936-213b17f265b2

Now, in the following example, let's add two blocks to our blockchain.

https://github.com/user-attachments/assets/f0047a3b-5864-47e9-8fdf-340a0504e0f8

# Transactions

Given what we have, let's introduce the concept of transactions to the blockchain.
Since a blockchain is a public database, we never want to store sensitive information like bank accounts or balances.
Transactions allow the transfer of tokens between addresses by consuming previous outputs and creating new ones. This is a major step in understanding how a blockchain manages its existing tokens.

## So... what is a transaction?

At its core, a transaction represents the transfer of value between participants in the blockchain. Instead of tracking balances like a traditional bank ledger, blockchain transactions follow an input-output model. Each transaction spends outputs from previous transactions and creates new outputs that can be used in future transactions.

## Transaction structure

A transaction contains simply encapsulates inputs and outputs:

```go
type Transaction struct {
	ID      []byte              // unique transaction identifier (hash)
	Inputs  []TransactionInput  // references to previous transaction outputs
	Outputs []TransactionOutput // new outputs containing tokens and recipient addresses
}
```

- `Inputs`: a slice that points to outputs from previous transactions (UTXOs)

  ```go
  type TransactionInput struct {
      ID        []byte // the ID of the transaction whose outputs will serve as inputs
      Output    int    // the index of the list of outputs of that transaction
      Signature string // ownership proof
  }
  ```

- `Outputs`: a slice containing new UTXOs, specifying the amount and the recipient's public key

  ```go
  type TransactionOutput struct {
      Value     int    // token amount
      PublicKey string // the recepient's address
  }
  ```

## Kickstart transactions

Like blocks, transactions need previous transactions to point to.
Similarly to the genesis blocks, a coinbase transaction referes to the first transaction of the blockchain.
This special transaction is also used to reward the first miner with tokens, and we can created it using the `CoinbaseTx()` method.

Once that's done, regular transactions can ensue by the help of the `newTransaction()` method. A new transaction block only occurs when the sender has enough tokens for that transaction.

## Practical scenario

Alice has 10 tokens and wants to send 7 tokens to Bob. In a blockchain system, this transaction is structured into **inputs** and **outputs**.

### Step 1: Identifying the Input

Alice owns a **previous unspent transaction output (UTXO)** worth **10 tokens**. She will use this as the input for her new transaction.

| **Transaction ID** | **Output Index** | **Value** | **Owner** |
| ------------------ | ---------------- | --------- | --------- |
| xyz789             | 0                | 10        | Alice     |

- **Transaction ID (`xyz789`)**: Identifies the transaction where Alice received the tokens.
- **Output Index (`0`)**: Specifies which output within that transaction she is spending.
- **Value (`10`)**: The amount available to spend.
- **Owner (`Alice`)**: The person who can spend this output.

### Step 2: Creating Outputs

Alice sends **7 tokens** to Bob, but since she started with **10 tokens**, she needs to get **3 tokens** back as change.

| **Value** | **Recipient**  |
| --------- | -------------- |
| 7         | Bob            |
| 3         | Alice (Change) |

- **Bob receives 7 tokens** in a new output.
- **Alice gets 3 tokens back** in another output (her change).

### Step 3: Structuring the Transaction

| **Transaction ID** | **Inputs**              | **Outputs**        |
| ------------------ | ----------------------- | ------------------ |
| abc123             | ID: xyz789, OutIndex: 0 | 7 → Bob, 3 → Alice |

- **Transaction ID (`abc123`)**: A unique identifier for the new transaction.
- **Inputs**: References Alice’s previous transaction (`xyz789`).
- **Outputs**:
  - 7 tokens to Bob.
  - 3 tokens returned to Alice.

# To do

- [ ] optimize performance given a large enough number of blocks by storing each block in its separate file
- [ ] have a parameter to sign the PoW to allow dynamical difficulty levels

```

```
