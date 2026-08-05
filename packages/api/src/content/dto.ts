import { IsOptional, IsString, IsNumber, IsEmail, Length } from 'class-validator'

// 内容统一模型：内容 = 标题 + 正文 + 可选媒体文件，不再有 type 分类。
export class UploadContentDto {
  @IsString() @Length(1, 200) title!: string
  @IsOptional() @IsString() content?: string
  @IsOptional() tags?: string | string[]
}

// 游客快速上传：title 必填；content（描述）与 file 至少一个（前端校验，后端兜底）。
// 已登录用户可省略 nickname/email。
export class QuickUploadDto {
  @IsString() @Length(1, 200) title!: string
  @IsOptional() @IsString() @Length(1, 50) nickname?: string
  @IsOptional() @IsEmail() email?: string
  @IsOptional() @IsString() content?: string
  @IsOptional() tags?: string | string[]
}

export class UpdateContentDto {
  @IsOptional() @IsString() title?: string
  @IsOptional() @IsString() content?: string
  @IsOptional() tags?: string | string[]
}

export class ListContentDto {
  @IsOptional() @IsNumber() page?: number
  @IsOptional() @IsNumber() page_size?: number
  @IsOptional() @IsString() tag?: string
  @IsOptional() @IsString() audit_status?: string
  @IsOptional() @IsString() keyword?: string
  @IsOptional() @IsString() sort_by?: string
  @IsOptional() @IsString() order?: string
}

export class ClaimDto {
  @IsOptional() @IsString() reason?: string
}
