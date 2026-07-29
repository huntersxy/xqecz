import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn } from 'typeorm'

@Entity('poll_votes')
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
