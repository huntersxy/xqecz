import { IsString, IsEmail, MinLength, MaxLength, Matches } from 'class-validator'

export class RegisterDto {
  @IsString()
  @MinLength(2)
  @MaxLength(32)
  @Matches(/^[\w\u4e00-\u9fff]+$/, { message: '用户名只能包含字母、数字、下划线或中文' })
  username!: string

  @IsEmail({}, { message: '请输入有效的邮箱地址' })
  email!: string

  @IsString()
  @MinLength(6)
  password!: string
}

export class LoginDto {
  @IsString()
  username!: string

  @IsString()
  password!: string
}

export class ChangePasswordDto {
  @IsString()
  @MinLength(6)
  oldPassword!: string

  @IsString()
  @MinLength(6)
  newPassword!: string
}

export class UpdateEmailDto {
  @IsEmail({}, { message: '请输入有效的邮箱地址' })
  email!: string
}
