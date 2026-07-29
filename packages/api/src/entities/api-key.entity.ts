import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn, UpdateDateColumn, DeleteDateColumn } from 'typeorm'

@Entity('api_keys')
export class ApiKey {
  @PrimaryGeneratedColumn({ type: 'bigint', unsigned: true })
  id!: number

  @Column({ type: 'bigint', unsigned: true })
  user_id!: number

  @Column({ type: 'varchar', length: 100 })
  name!: string

  @Column({ type: 'varchar', length: 10 })
  key_prefix!: string

  @Column({ type: 'varchar', length: 64, select: false })
  key_hash!: string

  @Column({ type: 'text', default: '[]' })
  permissions!: string

  @Column({ type: 'tinyint', default: 1 })
  is_active!: number

  @Column({ type: 'datetime', precision: 3, nullable: true })
  last_used_at?: Date

  @Column({ type: 'datetime', precision: 3, nullable: true })
  expires_at?: Date

  @CreateDateColumn({ type: 'datetime', precision: 3 })
  created_at!: Date

  @UpdateDateColumn({ type: 'datetime', precision: 3 })
  updated_at!: Date

  @DeleteDateColumn({ type: 'datetime', precision: 3 })
  deleted_at?: Date
}
