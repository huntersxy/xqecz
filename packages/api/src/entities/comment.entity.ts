import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn, UpdateDateColumn, DeleteDateColumn } from 'typeorm'

@Entity('comments')
export class Comment {
  @PrimaryGeneratedColumn({ type: 'bigint', unsigned: true })
  id!: number

  @Column({ type: 'bigint', unsigned: true })
  content_id!: number

  @Column({ type: 'bigint', unsigned: true })
  user_id!: number

  @Column({ type: 'text' })
  text!: string

  @Column({ type: 'bigint', unsigned: true, nullable: true })
  parent_id?: number

  @Column({ type: 'tinyint', default: 0 })
  is_banned!: number

  @CreateDateColumn({ type: 'datetime', precision: 3 })
  created_at!: Date

  @UpdateDateColumn({ type: 'datetime', precision: 3 })
  updated_at!: Date

  @DeleteDateColumn({ type: 'datetime', precision: 3 })
  deleted_at?: Date
}
