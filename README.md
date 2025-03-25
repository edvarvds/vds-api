# API VDS - Sistema de Gerenciamento de API

Sistema de gerenciamento de API com controle de acesso por domínio, cache e rate limiting.

## Características

- Controle de acesso por domínio (CORS)
- Cache com Redis
- Rate limiting
- Unificação de fontes de dados
- API RESTful

## Pré-requisitos

- Go 1.21 ou superior
- Redis
- Docker (opcional)

## Configuração

1. Clone o repositório:
```bash
git clone [seu-repositorio]
cd api_vds
```

2. Instale as dependências:
```bash
go mod download
```

3. Configure o Redis:
- Certifique-se de que o Redis está rodando
- Ajuste as configurações no arquivo `config/config.yaml` se necessário

4. Configure as variáveis de ambiente (opcional):
```bash
export REDIS_HOST=seu-redis-host
export REDIS_PORT=6379
export API_TOKEN=seu-token
```

## Executando o projeto

1. Em desenvolvimento:
```bash
go run main.go
```

2. Em produção:
```bash
go build
./api_vds
```

## Endpoints

### Consulta de CPF
```
GET /api/v1/cpf/:cpf
```

Exemplo de resposta:
```json
{
  "DADOS": {
    "cpf": "33512403840",
    "nome": "BRENDA REJANE COHEN",
    "nome_mae": "ANGELA REGINA COHEN",
    "data_nascimento": "1984-09-07 00:00:00",
    "sexo": "F"
  }
}
```

## Cache

O sistema utiliza Redis para cache com as seguintes características:
- Cache de CPF por 24 horas
- Cache automático de respostas
- Invalidação automática após expiração

## Segurança

- Controle de acesso por domínio
- Rate limiting por IP
- Tokens de autenticação para APIs externas
- Validação de entrada de dados

## Monitoramento

O sistema inclui logs para:
- Erros de cache
- Falhas de requisição
- Rate limiting excedido
- Acesso não autorizado

## Contribuindo

1. Faça um fork do projeto
2. Crie uma branch para sua feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request 