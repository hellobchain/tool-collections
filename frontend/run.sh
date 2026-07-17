cd frontend
# dev
npm install
RUN npm run build
npm run serve
# prod
npm install
npm run build -- --mode production