import { IsString, IsArray, IsOptional, IsNumber } from 'class-validator'

export class CreatePollDto {
  @IsString() title!: string
  @IsOptional() @IsString() description?: string
  @IsArray() options!: string[]
}

export class VoteDto {
  @IsNumber() option_index!: number
}
