import { spawn } from 'node:child_process'

const child = spawn(process.execPath, ['.output/server/index.mjs'], {
  env: {
    ...process.env,
    NITRO_HOST: '127.0.0.1',
    NITRO_PORT: '3001',
  },
  stdio: 'inherit',
})

function stopServer() {
  child.kill('SIGTERM')
}

process.on('SIGINT', stopServer)
process.on('SIGTERM', stopServer)
child.on('exit', (code) => process.exit(code ?? 0))
