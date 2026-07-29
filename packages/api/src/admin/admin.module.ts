import { Module } from '@nestjs/common'
import { TypeOrmModule } from '@nestjs/typeorm'
import { Content, User, Claim, CommentReport, Comment } from '../entities'
import { AdminService } from './admin.service'
import { AdminController } from './admin.controller'
import { ContentModule } from '../content/content.module'
import { CommentModule } from '../comment/comment.module'
import { AuthModule } from '../auth/auth.module'

@Module({
  imports: [TypeOrmModule.forFeature([Content, User, Claim, CommentReport, Comment]), ContentModule, CommentModule, AuthModule],
  controllers: [AdminController],
  providers: [AdminService],
})
export class AdminModule {}
