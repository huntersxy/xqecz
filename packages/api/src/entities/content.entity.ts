import { Entity, PrimaryGeneratedColumn, Column, CreateDateColumn, UpdateDateColumn, DeleteDateColumn } from 'typeorm'

@Entity('contents')
export class Content {
  @PrimaryGeneratedColumn({ type: 'bigint', unsigned: true })
  id!: number

  @Column({ type: 'varchar', length: 200 })
  title!: string

  @Column({ type: 'text', nullable: true, default: '' })
  content?: string

  // 媒体文件（展示即原文件，图片为 WebP，视频为原视频）；null = 纯文本内容。
  @Column({ type: 'varchar', length: 500, nullable: true })
  file_path?: string

  @Column({ type: 'bigint', default: 0 })
  file_size!: number

  @Column({ type: 'varchar', length: 500, nullable: true })
  thumb_path?: string

  @Column({ type: 'bigint', default: 0 })
  view_count!: number

  @Column({ type: 'bigint', unsigned: true })
  user_id!: number

  // 游客快速上传标识（user_id=0 时生效）：昵称对外展示，邮箱仅后台留档不外露。
  @Column({ type: 'varchar', length: 50, nullable: true })
  guest_nickname?: string

  @Column({ type: 'varchar', length: 254, nullable: true })
  guest_email?: string

  @Column({ type: 'text', default: '[]' })
  tags!: string // JSON array

  @Column({ type: 'varchar', length: 20, default: 'pending' })
  audit_status!: string

  @CreateDateColumn({ type: 'datetime', precision: 3 })
  created_at!: Date

  @UpdateDateColumn({ type: 'datetime', precision: 3 })
  updated_at!: Date

  @DeleteDateColumn({ type: 'datetime', precision: 3 })
  deleted_at?: Date
}
