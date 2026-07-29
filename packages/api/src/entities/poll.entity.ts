import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn, UpdateDateColumn, DeleteDateColumn } from 'typeorm'

@Entity('polls')
export class Poll {
  @PrimaryGeneratedColumn({ type: 'bigint', unsigned: true })
  id!: number

  @Column({ type: 'varchar', length: 200 })
  title!: string

  @Column({ type: 'text', nullable: true, default: '' })
  description?: string

  @Column({ type: 'text' })
  options!: string // JSON array

  @Column({ type: 'bigint', default: 0 })
  vote_count!: number

  @Column({ type: 'bigint', unsigned: true })
  user_id!: number

  @CreateDateColumn({ type: 'datetime', precision: 3 })
  created_at!: Date

  @UpdateDateColumn({ type: 'datetime', precision: 3 })
  updated_at!: Date

  @DeleteDateColumn({ type: 'datetime', precision: 3 })
  deleted_at?: Date
}
