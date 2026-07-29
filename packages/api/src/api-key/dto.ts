import { IsString, IsArray, IsOptional, IsBoolean } from 'class-validator'

export class CreateApiKeyDto {
  @IsOptional() @IsString() name?: string
  @IsOptional() @IsArray() permissions?: string[]
}

export class UpdateApiKeyDto {
  @IsOptional() @IsString() name?: string
  @IsOptional() @IsArray() permissions?: string[]
  @IsOptional() @IsBoolean() is_active?: boolean
}
