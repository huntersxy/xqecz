import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn, UpdateDateColumn, DeleteDateColumn } from 'typeorm'

@Entity('users')
export class User {
  @PrimaryGeneratedColumn({ type: 'bigint', unsigned: true })
  id!: number

  @Column({ type: 'varchar', length: 50, unique: true })
  username!: string

  @Column({ type: 'varchar', length: 254, nullable: true })
  email?: string

  @Column({ type: 'varchar', length: 255, select: false })
  password!: string

  @Column({ type: 'tinyint', default: 0 })
  is_admin!: number

  @Column({ type: 'tinyint', default: 0 })
  is_banned!: number

  @CreateDateColumn({ type: 'datetime', precision: 3 })
  created_at!: Date

  @UpdateDateColumn({ type: 'datetime', precision: 3 })
  updated_at!: Date

  @DeleteDateColumn({ type: 'datetime', precision: 3 })
  deleted_at?: Date
}
