import { Module } from '@nestjs/common'
import { TypeOrmModule } from '@nestjs/typeorm'
import { Content, User, Claim, ContentLike, ContentFavorite } from '../entities'
import { ContentService } from './content.service'
import { ContentController } from './content.controller'
import { AuthModule } from '../auth/auth.module'

@Module({
  imports: [TypeOrmModule.forFeature([Content, User, Claim, ContentLike, ContentFavorite]), AuthModule],
  controllers: [ContentController],
  providers: [ContentService],
  exports: [ContentService],
})
export class ContentModule {}
