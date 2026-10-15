FROM mcr.microsoft.com/playwright:v1.51.1-noble

WORKDIR /app

# Copy package definitions and install dependencies
COPY test/web/package.json ./
RUN npm install

# Copy test suite configuration and sources
COPY test/web/tsconfig.json test/web/playwright.config.ts ./
COPY test/web/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

COPY test/web/helpers ./helpers
COPY test/web/specs ./specs

ENV BASE_URL=http://127.0.0.1:8080
ENV PLAYWRIGHT_REPORT_DIR=/app/playwright-report

ENTRYPOINT ["/entrypoint.sh"]
