import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn } from 'typeorm'

@Entity('comment_reports')
export class CommentReport {
  @PrimaryGeneratedColumn({ type: 'bigint', unsigned: true })
  id!: number

  @Column({ type: 'bigint', unsigned: true })
  comment_id!: number

  @Column({ type: 'bigint', unsigned: true })
  user_id!: number

  @Column({ type: 'varchar', length: 255, default: '' })
  reason!: string

  @Column({ type: 'tinyint', default: 0 })
  handled!: number

  @CreateDateColumn({ type: 'datetime', precision: 3 })
  created_at!: Date
}
