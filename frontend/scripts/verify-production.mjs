import { readdir, readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { loadEnvFile } from 'node:process'

const outputDirectory = resolve(import.meta.dirname, '../.output')
const environmentFile = process.env.KAGARI_PRODUCTION_ENV_FILE || resolve(import.meta.dirname, '../../.env')
loadEnvFile(environmentFile)

const serverSecretNames = [
  'MYSQL_DSN',
  'MYSQL_PASSWORD',
  'MYSQL_ROOT_PASSWORD',
  'REDIS_PASSWORD',
  'ADMIN_PASSWORD',
  'QINIU_ACCESS_KEY',
  'QINIU_SECRET_KEY',
]
const configuredSecrets = serverSecretNames.map(name => {
  const value = process.env[name]
  if (!value || value.length < 8 || value.includes('replace-with-')) {
    throw new Error(`生产密钥验收需要 ${name} 的非占位配置值`)
  }
  return { name, value }
})

const files = await listFiles(outputDirectory)
for (const file of files) {
  const content = await readFile(file)
  if (content.includes(0)) {
    continue
  }
  const text = content.toString('utf8')
  for (const marker of serverSecretNames) {
    if (text.includes(marker)) {
      throw new Error(`公开构建产物 ${file} 包含服务端机密标记 ${marker}`)
    }
  }
  for (const secret of configuredSecrets) {
    if (text.includes(secret.value)) {
      throw new Error(`生产构建产物 ${file} 包含 ${secret.name} 的实际配置值`)
    }
  }
}

console.log(`Production verification passed: ${files.length} build files contain no server-secret markers or configured values.`)

async function listFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true })
  const nested = await Promise.all(entries.map(entry => {
    const path = resolve(directory, entry.name)
    if (entry.isSymbolicLink()) {
      return []
    }
    return entry.isDirectory() ? listFiles(path) : entry.isFile() ? [path] : []
  }))
  return nested.flat()
}
