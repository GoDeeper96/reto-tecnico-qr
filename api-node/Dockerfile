# API en Node: dependencias comunes y dos imágenes finales.
#   docker build --target local  -t api-node .   -> servidor HTTP (docker compose)
#   docker build --target lambda -t api-node .   -> AWS Lambda (runtime nodejs22.x)

FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev

FROM node:22-alpine AS test
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY src ./src
COPY test ./test
RUN npm test

FROM node:22-alpine AS local
WORKDIR /app
ENV NODE_ENV=production PORT=3000
COPY --from=deps /app/node_modules ./node_modules
COPY package.json ./
COPY src ./src
USER node
EXPOSE 3000
CMD ["node", "src/server.js"]

FROM public.ecr.aws/lambda/nodejs:22 AS lambda
ENV NODE_ENV=production
COPY --from=deps /app/node_modules ${LAMBDA_TASK_ROOT}/node_modules
COPY package.json ${LAMBDA_TASK_ROOT}/
COPY src ${LAMBDA_TASK_ROOT}/src
CMD ["src/lambda.handler"]
