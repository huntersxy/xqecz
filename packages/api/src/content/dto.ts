import { IsOptional, IsString, IsNumber, IsIn, IsEmail, Length } from 'class-validator'

// 2026-07-29 改造：上传"去分类"后，type 不再由调用方指定；后端按"是否有 file"自动设为 image / text。
// 历史数据 type 取值仍保留在 DB（兼容老数据 + 后端 service 内含迁移脚本一次性归并 video/link → text）。
export const CONTENT_TYPES = ['image', 'text', 'video', 'link'] as const

export class UploadContentDto {
  @IsString() @Length(1, 200) title!: string
  @IsOptional() @IsString() content?: string
  @IsOptional() tags?: string | string[]
}

// 游客快速上传：title 必填；content（描述）与 file 至少一个（前端校验，后端兜底）。
export class QuickUploadDto {
  @IsString() @Length(1, 200) title!: string
  @IsString() @Length(1, 50) nickname!: string
  @IsEmail() email!: string
  @IsOptional() @IsString() content?: string
  @IsOptional() tags?: string | string[]
}

export class UpdateContentDto {
  @IsOptional() @IsString() title?: string
  @IsOptional() @IsString() content?: string
  @IsOptional() @IsString() url?: string
  @IsOptional() tags?: string | string[]
}

export class ListContentDto {
  @IsOptional() @IsNumber() page?: number
  @IsOptional() @IsNumber() page_size?: number
  @IsOptional() @IsString() tag?: string
  @IsOptional() @IsString() type?: string
  @IsOptional() @IsString() audit_status?: string
  @IsOptional() @IsString() keyword?: string
  @IsOptional() @IsString() sort_by?: string
  @IsOptional() @IsString() order?: string
}

export class ClaimDto {
  @IsOptional() @IsString() reason?: string
}

// 一次性迁移脚本专用 DTO（管理员手动触发；仅允许迁移到合法值）。
export class MigrateTypesDto {
  @IsIn(['text']) to!: 'text'
}
