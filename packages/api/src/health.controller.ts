import { Controller, Get } from '@nestjs/common'

@Controller('health')
export class HealthController {
  @Get()
  check() {
    return { code: 200, message: 'ok', data: { ok: true } }
  }
}
