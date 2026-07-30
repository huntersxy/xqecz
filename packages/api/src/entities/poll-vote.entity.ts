import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn, Unique } from 'typeorm'

@Entity('poll_votes')
@Unique(['poll_id', 'user_id'])
@Unique(['poll_id', 'visitor_id'])
export class PollVote {
  @PrimaryGeneratedColumn({ type: 'bigint', unsigned: true })
  id!: number

  @Column({ type: 'bigint', unsigned: true })
  poll_id!: number

  @Column({ type: 'bigint', unsigned: true, nullable: true })
  user_id?: number

  @Column({ type: 'varchar', length: 64, nullable: true })
  visitor_id?: string

  @Column({ type: 'int' })
  option_index!: number

  @CreateDateColumn({ type: 'datetime', precision: 3 })
  created_at!: Date
}
