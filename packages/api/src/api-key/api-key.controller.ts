import { Controller, Get, Post, Put, Delete, Param, Body, UseGuards } from '@nestjs/common'
import { ApiKeyService } from './api-key.service'
import { AuthGuard } from '../guards/auth.guard'
import { CurrentUser } from '../decorators/current-user.decorator'
import { CreateApiKeyDto, UpdateApiKeyDto } from './dto'

@Controller('api-keys')
@UseGuards(AuthGuard)
export class ApiKeyController {
  constructor(private svc: ApiKeyService) {}

  @Post()
  async create(@Body() dto: CreateApiKeyDto, @CurrentUser('uid') uid: number) {
    const data = await this.svc.create(uid, dto.name, dto.permissions)
    return { code: 200, message: 'API 密钥已创建', data }
  }

  @Get()
  async list(@CurrentUser('uid') uid: number) {
    return { code: 200, message: 'ok', data: { list: await this.svc.list(uid) } }
  }

  @Put(':id')
  async update(@Param('id') id: string, @Body() dto: UpdateApiKeyDto, @CurrentUser('uid') uid: number) {
    const data = await this.svc.update(Number(id), uid, dto)
    return { code: 200, message: 'ok', data }
  }

  @Delete(':id')
  async delete(@Param('id') id: string, @CurrentUser('uid') uid: number) {
    await this.svc.delete(Number(id), uid)
    return { code: 200, message: '密钥已删除', data: null }
  }
}
