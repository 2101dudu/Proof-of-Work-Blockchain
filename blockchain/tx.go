package blockchain

type TransactionInput struct {
	ID        []byte // the ID of the transaction whose outputs will serve as inputs
	Output    int    // the index of the list of outputs of that transaction
	Signature string // ownership proof
}

type TransactionOutput struct {
	Value     int    // token amount
	PublicKey string // the recepient's address
}

func (txIn *TransactionInput) canUnlock(address string) bool {
	return txIn.Signature == address
}

func (txOut *TransactionOutput) canBeUnlocked(publicKey string) bool {
	return txOut.PublicKey == publicKey
}
