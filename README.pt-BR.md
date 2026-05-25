# AFI SDK

SDK para troca de tokens na rede Base via [Protocolo AFI](https://afi.run).

Disponível em **Node.js (TypeScript)** e **Go**.

---

## Instalação

```bash
# Go — instalar direto do GitHub
go get github.com/afi-run/sdk/go

# Node.js — clonar e instalar pelo caminho local
git clone https://github.com/afi-run/sdk.git
npm install ./sdk/nodejs
```

> **Node.js:** o npm não suporta instalação de subdiretório do GitHub.
> Quando o pacote for publicado no npm, a instalação será simplesmente `npm install @afi-run/sdk`.

---

## Início rápido

```typescript
// Node.js
import { AfiClient } from "@afi-run/sdk"

const client = new AfiClient({
  rpcUrl: "https://rpc.ankr.com/base/SUA_CHAVE",
  privateKey: "0xSUA_CHAVE_PRIVADA",
})

// Recomendado: obter cotação primeiro, depois executar
const quote = await client.getQuote({
  tokenIn:  "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC
  tokenOut: "0x4200000000000000000000000000000000000006", // WETH
  amountIn: 1000_000000n,  // 1000 USDC (raw wei, 6 casas decimais)
  slippage: 0.5,           // 0.5%
})

console.log("Valor estimado:", quote.amountOutWei)
console.log("Mínimo garantido:", quote.minOutWei) // aplicado no contrato

const result = await client.executeSwap(quote)
console.log("Hash da tx:", result.txHash)
console.log("Recebido:  ", result.amountOut, "wei WETH")
```

```go
// Go
client, _ := afi.NewClient(afi.Config{
    RPCURL:     "https://rpc.ankr.com/base/SUA_CHAVE",
    PrivateKey: "SUA_CHAVE_PRIVADA",
})
defer client.Close()

amountIn, _ := afi.ParseUnits("1000", 6) // 1000 USDC

quote, _ := client.GetQuote(ctx, afi.SwapParams{
    TokenIn:  common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
    TokenOut: afi.WETH,
    AmountIn: amountIn,
    Slippage: 0.5,
})

result, _ := client.ExecuteSwap(ctx, quote)
fmt.Println("Tx:", result.TxHash.Hex())
fmt.Println("Recebido:", afi.FormatUnits(result.AmountOut, 18), "WETH")
```

---

## Referência da API

### `getTokens()` — listar tokens disponíveis

```typescript
const tokens = await client.getTokens()
// Token[] — tokens ativos na rede Base

// Estrutura de Token:
// {
//   address:  "0x833589..."
//   symbol:   "USDC"
//   decimals: 6
//   active:   true
// }
```

Consulta `GET https://rpc.afi.run/info` e retorna os tokens suportados na Base.
Use isso na inicialização para permitir que o usuário escolha entre endereços válidos.
Não requer chave privada nem interação com a blockchain.

```go
// Go
tokens, err := client.GetTokens(ctx)
for _, t := range tokens {
    fmt.Printf("%s → %s\n", t.Symbol, t.Address.Hex())
}
```

---

### `getQuote(params)` — obter cotação de preço

```typescript
const quote = await client.getQuote({
  tokenIn:  "0x...",               // endereço do token de entrada
  tokenOut: "0x...",               // endereço do token de saída
  amountIn: parseUnits("1000", 6), // 1000 USDC — use parseUnits para evitar raw wei
  slippage: 0.5,                   // percentual, ex: 0.5 = 0.5%
})
```

**Somente leitura** — nenhuma transação é enviada. Seguro para chamar com frequência.
Internamente:

1. Lê `decimals()` do contrato do token de entrada
2. Lê `feeBps()` do contrato AFI (em tempo real — a taxa pode mudar)
3. Chama `POST https://rpc.afi.run/quoter` com sua URL RPC

**Campos da cotação:**

| Campo | Tipo | Descrição |
|---|---|---|
| `amountInWei` | `bigint` | Valor exato a aprovar e enviar |
| `amountOutWei` | `bigint` | Saída estimada (informativo) |
| `minOutWei` | `bigint` | Saída mínima com slippage — aplicada no contrato |
| `steps` | `Hex` | Rota codificada passada para `Afi.swap()` — não modificar |
| `path` | `Address[]` | Endereços dos tokens no caminho da rota |
| `slippage` | `number` | Percentual de slippage aplicado |
| `feeBps` | `number` | Taxa do protocolo no momento da cotação (basis points) |

**Importante:** `minOutWei` nunca é 0. O SDK rejeita cotações com saída mínima zero.

---

### `executeSwap(quote)` — executar uma cotação prévia

```typescript
const result = await client.executeSwap(quote)
```

Recebe um `Quote` retornado por `getQuote()` e executa o fluxo completo:

```
1. assertBalance  — verifica saldo do tokenIn ≥ amountInWei
2. approve        — aprova exatamente amountInWei para o contrato AFI
                    (ignorado se o allowance já é suficiente)
3. simulate       — executa eth_call antes de enviar — lança erro se a tx reverteria
4. swap           — envia a transação com estimativa de gas × 1.2
5. parse evento   — aguarda recibo e lê valores reais do evento SwapExecuted
```

**Por que simular primeiro?** Se a troca reverteria (ex: preço passou do `minOut`),
`SimulationFailedError` é lançado antes de qualquer gas ser gasto.

**Campos do resultado:**

| Campo | Tipo | Descrição |
|---|---|---|
| `txHash` | `Hex` | Hash da transação |
| `blockNumber` | `bigint` | Bloco onde a troca foi confirmada |
| `amountIn` | `bigint` | Entrada real do evento on-chain `SwapExecuted` |
| `amountOut` | `bigint` | Saída real do evento on-chain `SwapExecuted` |
| `gasUsed` | `bigint` | Gas consumido pela transação |

---

### `swap(params)` — atalho: cotação + execução em uma chamada

```typescript
const result = await client.swap({
  tokenIn:  "0x...",
  tokenOut: "0x...",
  amountIn: 500_000000n,
  slippage: 1.0,
})
```

Equivalente a `const q = await getQuote(params); return executeSwap(q)`.

Use para bots e scripts. Para apps com interface de usuário, prefira o padrão
`getQuote` → exibir preço → `executeSwap` para que o usuário confirme antes.

---

### `approve(tokenIn, amountWei)` — aprovar apenas

```typescript
const txHash = await client.approve(tokenIn, quote.amountInWei)
// Retorna null se o allowance já era suficiente (sem tx enviada)
```

Aprova exatamente `amountWei` — sem excesso — para o contrato AFI.

`executeSwap()` chama isso automaticamente. Use diretamente apenas se seu app
precisar de dois prompts de carteira separados (aprovar e depois trocar).

**Detalhes de segurança:**
- Verifica allowance on-chain primeiro — ignora se já suficiente
- Reseta para 0 antes de re-aprovar para tokens estilo USDT que exigem isso
- Reverifica allowance on-chain após a tx de aprovação confirmar

---

### `getFeeBps()` — ler taxa atual do protocolo

```typescript
const feeBps = await client.getFeeBps()
// ex: 35 → 0.35%
```

Lê `feeBps` diretamente do contrato AFI. A taxa pode mudar e já está incluída
em todo objeto `Quote` retornado por `getQuote()`.

---

### `parseUnits(amount, decimals)` / `formatUnits(amount, decimals)` — helpers de unidade

```typescript
import { parseUnits, formatUnits } from "@afi-run/sdk"

// String legível → raw wei (bigint) — use como amountIn
parseUnits("1000", 6)          // 1000_000000n  (1000 USDC)
parseUnits("1.5", 6)           // 1_500000n
parseUnits("0.5", 18)          // 500000000000000000n  (0.5 WETH)

// Raw wei (bigint) → string legível — use para exibir valores
formatUnits(1000_000000n, 6)   // "1000"
formatUnits(1_500000n, 6)      // "1.5"
formatUnits(500000000000000000n, 18) // "0.5"
```

```go
// Go
wei, err := afi.ParseUnits("1000", 6)   // big.Int 1000_000000
str := afi.FormatUnits(wei, 6)          // "1000"
```

Esses helpers permitem trabalhar com os valores que o usuário digita em vez de raw wei:

```typescript
// Sem helpers
const quote = await client.getQuote({
  amountIn: 1000_000000n,  // precisa saber que USDC tem 6 decimais
  ...
})

// Com helpers
const quote = await client.getQuote({
  amountIn: parseUnits("1000", 6),  // legível
  ...
})
```

---

## Aprovação: por que sempre exata?

O SDK aprova exatamente o valor da cotação — nunca mais. Isso significa:

- Mesmo que o contrato AFI fosse comprometido, um atacante só poderia gastar o
  valor que você já ia gastar naquele swap específico
- Se você trocar com frequência, uma tx de aprovação será enviada por troca
- `executeSwap()` ignora a aprovação se o allowance existente já for suficiente

---

## Garantias de segurança

| Risco | Como o SDK trata |
|---|---|
| Bypass de slippage | `minOut` sempre vem da API — nunca definido como 0 |
| Aprovação excessiva | Sempre aprova exatamente `amountInRaw` da cotação |
| Tokens estilo USDT | Reseta allowance para 0 antes de re-aprovar se necessário |
| Tx que reverteria | Simulação `eth_call` roda antes de cada swap — falha rápido |
| Race condition (allow vs swap) | Allowance reverificado on-chain após confirmação |
| Subestimativa de gas | Gas estimado on-chain e multiplicado por 1.2 |
| ETH nativo passado | Não suportado — use WETH: `0x4200000000000000000000000000000000000006` |

---

## Tratamento de erros

### Node.js

```typescript
import {
  InsufficientBalanceError,
  SimulationFailedError,
  QuoteError,
  ApprovalError,
  SwapRevertedError,
} from "@afi-run/sdk"

try {
  const result = await client.executeSwap(quote)
} catch (e) {
  if (e instanceof InsufficientBalanceError) {
    // Usuário não tem saldo suficiente do tokenIn
    console.log("Saldo:    ", e.balance)   // bigint, raw wei
    console.log("Necessário:", e.required) // bigint, raw wei
    console.log("Token:    ", e.token)     // endereço

  } else if (e instanceof SimulationFailedError) {
    // Swap reverteria — nenhuma tx foi enviada, sem gas gasto
    console.log("Motivo:", e.reason)       // string de revert decodificada
    console.log("Dados: ", e.revertData)   // bytes raw (opcional)

  } else if (e instanceof QuoteError) {
    // API de cotação retornou erro (ex: sem rota encontrada)
    console.log(e.message)

  } else if (e instanceof ApprovalError) {
    // Transação de aprovação do token falhou
    console.log(e.message)

  } else if (e instanceof SwapRevertedError) {
    // Transação de swap reverteu on-chain
    console.log("Motivo:", e.reason)
  }
}
```

### Go

```go
import "errors"

result, err := client.ExecuteSwap(ctx, quote)
if err != nil {
    var afiErr *afi.AfiError
    if errors.As(err, &afiErr) {
        switch afiErr.Code {
        case "INSUFFICIENT_BALANCE":
            fmt.Println("Saldo insuficiente")
        case "SIMULATION_FAILED":
            // Nenhuma tx foi enviada
            fmt.Println("Reverteria:", afiErr.Message)
        case "QUOTE_FAILED":
            fmt.Println("Sem rota encontrada:", afiErr.Message)
        case "APPROVAL_FAILED":
            fmt.Println("Aprovação falhou:", afiErr.Message)
        case "SWAP_REVERTED":
            fmt.Println("Swap reverteu:", afiErr.Message)
        }
        return
    }
    log.Fatal(err) // erro inesperado
}
```

---

## Constantes

| Nome | Valor |
|---|---|
| Contrato AFI (Base) | `0xB8cC65321d169D55b93b4402D795701c6B308ce4` |
| WETH (Base) | `0x4200000000000000000000000000000000000006` |
| API Quoter | `https://rpc.afi.run/quoter` |
| API Info | `https://rpc.afi.run/info` |
| Chain ID | `8453` |

```typescript
import { AFI_ADDRESS, WETH } from "@afi-run/sdk"
```

```go
afi.AfiAddress // common.Address
afi.WETH       // common.Address
```

---

## Exemplos

### Node.js

| Arquivo | O que demonstra |
|---|---|
| `examples/nodejs/1-list-tokens.ts` | Listar tokens disponíveis |
| `examples/nodejs/2-get-quote.ts` | Obter e inspecionar cotação |
| `examples/nodejs/3-execute-swap.ts` | Cotação → revisão → execução (recomendado) |
| `examples/nodejs/4-full-flow.ts` | Atalho em uma chamada |
| `examples/nodejs/5-approve-only.ts` | Aprovar e trocar em etapas separadas |

```bash
cd nodejs
npm install
npx ts-node ../examples/nodejs/1-list-tokens.ts
```

### Go

| Diretório | O que demonstra |
|---|---|
| `examples/go/list-tokens/` | Listar tokens disponíveis |
| `examples/go/get-quote/` | Obter e inspecionar cotação |
| `examples/go/execute-swap/` | Cotação → revisão → execução (recomendado) |
| `examples/go/full-flow/` | Atalho em uma chamada |
| `examples/go/approve-only/` | Aprovar e trocar em etapas separadas |

```bash
cd examples/go
go mod tidy
go run ./list-tokens
go run ./get-quote
go run ./execute-swap
```

---

## Build a partir do código-fonte

```bash
# Node.js
cd nodejs
npm install
npm run build        # gera arquivos em dist/
npm run typecheck    # verificação de tipos apenas
npm test             # rodar testes

# Go
cd go
go mod tidy
go build ./...
go test ./...
```

---

## Versões mínimas

| Ambiente | Versão |
|---|---|
| Node.js | ≥ 24.0.0 (LTS) |
| Go | 1.26.3 |
