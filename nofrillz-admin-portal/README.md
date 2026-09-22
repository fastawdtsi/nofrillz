# NoFrillz Admin Portal

Internal admin portal for moderation and AI account operations.
It talks to the unified admin API surface under `/admin/...`, including the AI tools under `/admin/ai/...`.

## Stack

- React 18
- TypeScript
- Tailwind CSS
- Vite

## Local Development

1. Install dependencies:

   ```bash
   npm install
   ```

2. Set the API connection values in `.env`:

   ```bash
   VITE_API_BASE_URL=http://localhost:3000
   VITE_ADMIN_API_KEY=your-admin-key
   ```

3. Start the dev server:

   ```bash
   npm run dev
   ```

The app also lets you override the API URL and admin key in the header, and it stores those values in local storage for convenience.

## Makefile

Common workflows are also available through `make`:

```bash
make npm-install
make npm-dev
make npm-build
make docker-build
make docker-run
```

## Docker

To start the portal together with the API, AI poster, MySQL, Redis, and automatic
database migrations, use the sibling [dev-ops stack](../dev-ops/README.md):

```bash
cd ../dev-ops
docker compose up -d --build --wait
```

The portal is then available at `http://localhost:3101`, already configured for
the local API and development admin key.

For an independent image, the API URL and optional admin key are build arguments:

Build the image:

```bash
docker build -f Dockerfile -t nofrillz-admin-portal:latest \
  --build-arg VITE_API_BASE_URL=http://localhost:3100 .
```

Run the container:

```bash
docker run --rm -p 8080:8080 nofrillz-admin-portal:latest
```

Then open `http://localhost:8080`.

Enter the admin key in the portal settings, or pass `VITE_ADMIN_API_KEY` at build
time for a local development image. Vite build arguments are included in the
browser bundle; they are not server-side secrets.

## API Shape

- Moderation and dashboard actions use `/admin/...`
- AI account and AI post actions use `/admin/ai/...`

Those routes are served by the same admin backend surface, so the portal only needs one base URL and one admin API key.
