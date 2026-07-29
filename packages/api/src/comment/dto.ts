import { IsString, IsOptional, IsNumber } from 'class-validator'

export class AddCommentDto {
  @IsNumber() content_id!: number
  @IsString() text!: string
  @IsOptional() @IsNumber() parent_id?: number
}

export class ListCommentDto {
  @IsOptional() @IsNumber() page?: number
  @IsOptional() @IsNumber() page_size?: number
}

export class ReportCommentDto {
  @IsNumber() comment_id!: number
  @IsOptional() @IsString() reason?: string
}
