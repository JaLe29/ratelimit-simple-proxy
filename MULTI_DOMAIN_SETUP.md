# Nasazení Proxy na Více Domén

Tento dokument popisuje, jak nasadit proxy na více domén s podporou dynamických redirect URL pro Google OAuth.

## Problém s Původní Implementací

Původní implementace měla následující omezení:
- Pevně nastavená redirect URL: `https://auth.jale.cz/auth/callback`
- Jeden auth domain pro všechny domény
- Google OAuth povoluje pouze omezený počet redirect URLs

## Nové Řešení

### 1. Dynamické Redirect URL

Každá doména může mít svou vlastní auth konfiguraci:

```yaml
rateLimits:
  "auto.cz":
    destination: "http://auto.app:2000"
    requests: 50
    perSecond: 1
    allowedEmails:
      - "jakubloffelmann@gmail.com"
    # Auth configuration for this domain
    auth:
      domain: "auth.auto.cz"  # Custom auth domain
      redirectUrl: "https://auth.auto.cz/auth/callback"
```

### 2. Konfigurace pro Různé Scénáře

#### Scénář 1: Všechny domény používají stejný auth domain
```yaml
googleAuth:
  enabled: true
  clientId: "your-client-id"
  clientSecret: "your-client-secret"
  authDomain: "auth.jale.cz"
  redirectUrl: "https://auth.jale.cz/auth/callback"

rateLimits:
  "domain1.cz":
    destination: "http://app1:2000"
    auth:
      domain: "auth.jale.cz"  # Používá default auth domain
      redirectUrl: "https://auth.jale.cz/auth/callback"

  "domain2.cz":
    destination: "http://app2:2000"
    auth:
      domain: "auth.jale.cz"  # Používá default auth domain
      redirectUrl: "https://auth.jale.cz/auth/callback"
```

#### Scénář 2: Každá doména má svůj auth domain
```yaml
googleAuth:
  enabled: true
  clientId: "your-client-id"
  clientSecret: "your-client-secret"
  authDomain: "auth.jale.cz"  # Fallback auth domain
  redirectUrl: "https://auth.jale.cz/auth/callback"

rateLimits:
  "auto.cz":
    destination: "http://auto.app:2000"
    auth:
      domain: "auth.auto.cz"
      redirectUrl: "https://auth.auto.cz/auth/callback"

  "shop.cz":
    destination: "http://shop.app:2000"
    auth:
      domain: "auth.shop.cz"
      redirectUrl: "https://auth.shop.cz/auth/callback"
```

#### Scénář 3: Smíšený přístup
```yaml
googleAuth:
  enabled: true
  clientId: "your-client-id"
  clientSecret: "your-client-secret"
  authDomain: "auth.jale.cz"
  redirectUrl: "https://auth.jale.cz/auth/callback"

rateLimits:
  "domain1.cz":
    destination: "http://app1:2000"
    # Používá default auth konfiguraci (není specifikováno auth)

  "domain2.cz":
    destination: "http://app2:2000"
    auth:
      domain: "auth.domain2.cz"
      redirectUrl: "https://auth.domain2.cz/auth/callback"
```

## Google OAuth Konfigurace

### 1. Google Console Nastavení

Pro každou doménu musíte přidat redirect URL do Google OAuth konfigurace:

1. Jděte do [Google Cloud Console](https://console.cloud.google.com/)
2. Vyberte váš projekt
3. Jděte do "APIs & Services" > "Credentials"
4. Upravte OAuth 2.0 Client ID
5. Přidejte všechny redirect URLs do "Authorized redirect URIs":
   - `https://auth.jale.cz/auth/callback`
   - `https://auth.auto.cz/auth/callback`
   - `https://auth.shop.cz/auth/callback`
   - atd.

### 2. DNS Nastavení

Pro každou auth doménu nastavte DNS záznamy:

```bash
# Pro auth.auto.cz
auth.auto.cz.    IN  A   YOUR_PROXY_IP

# Pro auth.shop.cz
auth.shop.cz.    IN  A   YOUR_PROXY_IP
```

### 3. SSL Certifikáty

Zajistěte, že máte SSL certifikáty pro všechny auth domény:

```bash
# Let's Encrypt pro více domén
certbot certonly --nginx -d auth.jale.cz -d auth.auto.cz -d auth.shop.cz
```

## Nasazení

### 1. Docker Compose

```yaml
version: '3.8'
services:
  proxy:
    build: .
    ports:
      - "80:8080"
      - "443:8443"
    volumes:
      - ./config.yaml:/app/config.yaml
    environment:
      - SSL_CERT_PATH=/etc/ssl/certs
      - SSL_KEY_PATH=/etc/ssl/private
```

### 2. Nginx Reverse Proxy (volitelné)

Pokud používáte Nginx jako reverse proxy:

```nginx
server {
    listen 443 ssl;
    server_name auth.auto.cz;

    ssl_certificate /etc/ssl/certs/auth.auto.cz.crt;
    ssl_certificate_key /etc/ssl/private/auth.auto.cz.key;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Testování

### 1. Test jednotlivých domén

```bash
# Test auto.cz
curl -H "Host: auto.cz" http://localhost:8080/

# Test auth.auto.cz
curl -H "Host: auth.auto.cz" http://localhost:8080/auth/callback
```

### 2. Test OAuth flow

1. Otevřete `https://auto.cz` v prohlížeči
2. Měli byste být přesměrováni na login stránku
3. Po přihlášení byste měli být přesměrováni na `https://auth.auto.cz/auth/callback`
4. Nakonec byste měli být přesměrováni zpět na `https://auto.cz`

## Troubleshooting

### 1. Chyba "Invalid redirect_uri"

Zkontrolujte, že redirect URL v Google Console odpovídá té v konfiguraci.

### 2. Chyba "Host not found"

Zkontrolujte, že doména je správně nakonfigurována v `rateLimits`.

### 3. SSL chyby

Zkontrolujte, že SSL certifikáty jsou správně nainstalovány pro všechny auth domény.

## Výhody Nového Řešení

1. **Flexibilita**: Každá doména může mít svou vlastní auth konfiguraci
2. **Škálovatelnost**: Snadné přidání nových domén
3. **Bezpečnost**: Izolované auth domény pro různé aplikace
4. **Spolehlivost**: Fallback na default auth konfiguraci