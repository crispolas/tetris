# Tetris

Clone do clássico Tetris desenvolvido em Go utilizando a biblioteca Ebiten.

## Como jogar

A forma mais fácil é baixar a versão mais recente na página de **Releases**:

1. Acesse a aba **Releases** deste repositório.
2. Baixe o arquivo `tetris.exe`.
3. Execute o arquivo e jogue.

Não é necessário instalar Go ou compilar o projeto.

## Controles

| Tecla  | Ação              |
| ------ | ----------------- |
| ← →    | Mover peça        |
| ↑      | Rotacionar peça   |
| ↓      | Acelerar queda    |
| Espaço | Queda instantânea |
| R      | Reiniciar partida |
| Esc    | Sair do jogo      |

## Recursos

* Música de fundo
* Efeitos sonoros
* Sistema de pontuação
* Limpeza de linhas
* Tela de Game Over

## Compilando o projeto

Caso queira executar a partir do código-fonte:

```bash
git clone https://github.com/SEU_USUARIO/tetris.git
cd tetris
go run .
```

Ou gerar um executável:

```bash
go build -o tetris.exe .
```

## Tecnologias utilizadas

* Go
* Ebiten

## Licença

Projeto desenvolvido para fins de estudo e aprendizado.
