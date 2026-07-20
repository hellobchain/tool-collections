module.exports = {
  productionSourceMap: false,
  configureWebpack: {
    devtool: false
  },
  devServer: {
    port: 8080,
    proxy: {
      '/textparse': {
        target: 'http://localhost:80',
        changeOrigin: true
      },
      '/weekly-assistant': {
        target: 'http://localhost:80',
        changeOrigin: true
      },
      '/md2docx': {
        target: 'http://localhost:80',
        changeOrigin: true
      }
    }
  }
}