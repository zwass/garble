package literals

import (
	"go/ast"
	"go/token"
	"math"
	mathrand "math/rand"

	ah "mvdan.cc/garble/internal/asthelper"
)

type swap struct{}

// check that the obfuscator interface is implemented
var _ obfuscator = swap{}

func getIndexType(dataLen int) string {
	switch {
	case dataLen <= math.MaxUint8:
		return "byte"
	case dataLen <= math.MaxUint16:
		return "uint16"
	case dataLen <= math.MaxUint32:
		return "uint32"
	default:
		return "uint64"
	}
}

func positionsToSlice(data []int) *ast.CompositeLit {
	arr := &ast.CompositeLit{
		Type: &ast.ArrayType{
			Len: &ast.Ellipsis{}, // Performance optimization
			Elt: ah.Ident(getIndexType(len(data))),
		},
		Elts: []ast.Expr{},
	}
	for _, data := range data {
		arr.Elts = append(arr.Elts, ah.IntLit(data))
	}
	return arr
}

// Generates a random even swap count based on the length of data
func generateSwapCount(dataLen int) int {
	swapCount := dataLen

	maxExtraPositions := dataLen / 2 // Limit the number of extra positions to half the data length
	if maxExtraPositions > 1 {
		swapCount += mathrand.Intn(maxExtraPositions)
	}
	if swapCount%2 != 0 { // Swap count must be even
		swapCount++
	}
	return swapCount
}

func (x swap) obfuscate(data []byte) *ast.BlockStmt {
	swapCount := generateSwapCount(len(data))
	shiftKey := byte(mathrand.Intn(math.MaxUint8))

	positions := genRandIntSlice(len(data), swapCount)
	for i := len(positions) - 2; i >= 0; i -= 2 {
		// Generate local key for xor based on random key and byte position
		localKey := byte(i) + byte(positions[i]^positions[i+1]) + shiftKey
		// Swap bytes from i+1 to i and xor using local key
		data[positions[i]], data[positions[i+1]] = data[positions[i+1]]^localKey, data[positions[i]]^localKey
	}

	return ah.BlockStmt(
		&ast.AssignStmt{
			Lhs: []ast.Expr{ah.Ident("data")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{ah.DataToByteSlice(data)},
		},
		&ast.AssignStmt{
			Lhs: []ast.Expr{ah.Ident("positions")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{positionsToSlice(positions)},
		},
		&ast.ForStmt{
			Init: &ast.AssignStmt{
				Lhs: []ast.Expr{ah.Ident("i")},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{ah.IntLit(0)},
			},
			Cond: &ast.BinaryExpr{
				X:  ah.Ident("i"),
				Op: token.LSS,
				Y:  ah.IntLit(len(positions)),
			},
			Post: &ast.AssignStmt{
				Lhs: []ast.Expr{ah.Ident("i")},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{ah.IntLit(2)},
			},
			Body: ah.BlockStmt(
				&ast.AssignStmt{
					Lhs: []ast.Expr{ah.Ident("localKey")},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{&ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X:  ah.CallExpr(ah.Ident("byte"), ah.Ident("i")),
							Op: token.ADD,
							Y: ah.CallExpr(ah.Ident("byte"), &ast.BinaryExpr{
								X:  ah.IndexExpr("positions", ah.Ident("i")),
								Op: token.XOR,
								Y: ah.IndexExpr("positions", &ast.BinaryExpr{
									X:  ah.Ident("i"),
									Op: token.ADD,
									Y:  ah.IntLit(1),
								}),
							}),
						},
						Op: token.ADD,
						Y:  ah.IntLit(int(shiftKey)),
					}},
				},
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						ah.IndexExpr("data", ah.IndexExpr("positions", ah.Ident("i"))),
						ah.IndexExpr("data", ah.IndexExpr("positions", &ast.BinaryExpr{
							X:  ah.Ident("i"),
							Op: token.ADD,
							Y:  ah.IntLit(1),
						})),
					},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{
						&ast.BinaryExpr{
							X: ah.IndexExpr("data", ah.IndexExpr("positions", &ast.BinaryExpr{
								X:  ah.Ident("i"),
								Op: token.ADD,
								Y:  ah.IntLit(1),
							})),
							Op: token.XOR,
							Y:  ah.Ident("localKey"),
						},
						&ast.BinaryExpr{
							X:  ah.IndexExpr("data", ah.IndexExpr("positions", ah.Ident("i"))),
							Op: token.XOR,
							Y:  ah.Ident("localKey"),
						},
					},
				},
			),
		},
	)
}
