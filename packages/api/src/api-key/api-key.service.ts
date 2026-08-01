import { Injectable, NotFoundException } from '@nestjs/common'
import { InjectRepository } from '@nestjs/typeorm'
import { Repository } from 'typeorm'
import { ApiKey } from '../entities'
import { createHash, randomBytes } from 'crypto'

@Injectable()
export class ApiKeyService {
  constructor(@InjectRepository(ApiKey) private repo: Repository<ApiKey>) {}

  private generate() {
    const raw = 'xq_' + randomBytes(24).toString('hex')
    // key_prefix 列为 varchar(10)，前缀取 10 位（如 xq_abcd1234）
    return { raw, prefix: raw.slice(0, 10), hash: createHash('sha256').update(raw).digest('hex') }
  }

  private decorate(row: ApiKey) {
    return {
      id: row.id, name: row.name, key_prefix: row.key_prefix,
      permissions: JSON.parse(row.permissions || '[]'),
      is_active: !!row.is_active, last_used_at: row.last_used_at ?? null, created_at: row.created_at,
    }
  }

  async create(userId: number, name?: string, permissions?: string[]) {
    const { raw, prefix, hash } = this.generate()
    const row = this.repo.create({ user_id: userId, name: name || 'default', key_prefix: prefix, key_hash: hash, permissions: JSON.stringify(permissions || []), is_active: 1 })
    const saved = await this.repo.save(row)
    return { ...this.decorate(saved), key: raw }
  }

  async list(userId: number) {
    const rows = await this.repo.find({ where: { user_id: userId }, order: { id: 'DESC' } })
    return rows.map((r) => this.decorate(r))
  }

  async update(id: number, userId: number, fields: { name?: string; permissions?: string[]; is_active?: boolean }) {
    const row = await this.repo.findOne({ where: { id, user_id: userId } })
    if (!row) throw new NotFoundException('密钥不存在')
    const update: any = {}
    if (fields.name) update.name = fields.name
    if (fields.permissions) update.permissions = JSON.stringify(fields.permissions)
    if (fields.is_active !== undefined) update.is_active = fields.is_active ? 1 : 0
    if (Object.keys(update).length) await this.repo.update(id, update)
    const updated = await this.repo.findOne({ where: { id } })!
    return this.decorate(updated!)
  }

  async delete(id: number, userId: number) {
    const row = await this.repo.findOne({ where: { id, user_id: userId } })
    if (!row) throw new NotFoundException('密钥不存在')
    await this.repo.softDelete(id)
  }
}
