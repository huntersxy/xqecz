import { IsString, IsOptional, IsNumber, IsIn, IsBoolean } from 'class-validator'

export class AuditDto {
  @IsString() @IsIn(['approved', 'rejected']) status!: string
  @IsOptional() @IsString() remark?: string
}

export class AuthorDto {
  @IsNumber() user_id!: number
}

export class RoleDto {
  @IsBoolean() is_admin!: boolean
}

export class BanDto {
  @IsBoolean() is_banned!: boolean
}

export class HandleClaimDto {
  @IsString() @IsIn(['approve', 'reject']) action!: 'approve' | 'reject'
  @IsOptional() @IsString() remark?: string
}
