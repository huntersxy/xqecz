import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn, Unique } from 'typeorm'

@Entity('content_likes')
@Unique(['content_id', 'user_id'])
export class ContentLike {
  @PrimaryGeneratedColumn({ type: 'bigint', unsigned: true })
  id!: number

  @Column({ type: 'bigint', unsigned: true })
  content_id!: number

  @Column({ type: 'bigint', unsigned: true })
  user_id!: number

  @CreateDateColumn({ type: 'datetime', precision: 3 })
  created_at!: Date
}
