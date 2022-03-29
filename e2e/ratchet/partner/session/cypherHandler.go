package session

type CypherHandler interface {
	AddKey(k *Cypher)
	DeleteKey(k *Cypher)
}
