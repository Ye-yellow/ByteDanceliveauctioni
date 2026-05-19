package data

// Data is the shared infrastructure dependency holder. It will own MySQL,
// Redis, message stream, and external AI clients when the prototype graduates
// from memory mode.
type Data struct{}
