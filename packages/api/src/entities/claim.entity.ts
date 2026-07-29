import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn, UpdateDateColumn } from 'typeorm'

@Entity('claims')
export class Claim {
  @PrimaryGeneratedColumn({ type: 'bigint', unsigned: true })
  id!: number

  @Column({ type: 'bigint', unsigned: true })
  content_id!: number

  @Column({ type: 'bigint', unsigned: true })
  user_id!: number

  @Column({ type: 'text', nullable: true, default: '' })
  reason?: string

  @Column({ type: 'varchar', length: 20, default: 'pending' })
  status!: string

  @Column({ type: 'bigint', unsigned: true, nullable: true })
  approved_by?: number

  @Column({ type: 'text', nullable: true, default: '' })
  remark?: string

  @CreateDateColumn({ type: 'datetime', precision: 3 })
  created_at!: Date

  @UpdateDateColumn({ type: 'datetime', precision: 3 })
  updated_at!: Date
}
