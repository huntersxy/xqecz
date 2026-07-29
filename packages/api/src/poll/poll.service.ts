import { Injectable, NotFoundException, ForbiddenException, BadRequestException } from '@nestjs/common'
import { InjectRepository } from '@nestjs/typeorm'
import { Repository } from 'typeorm'
import { Poll, PollVote, User } from '../entities'
import { randomBytes } from 'crypto'

@Injectable()
export class PollService {
  constructor(
    @InjectRepository(Poll) private pollRepo: Repository<Poll>,
    @InjectRepository(PollVote) private voteRepo: Repository<PollVote>,
    @InjectRepository(User) private userRepo: Repository<User>,
  ) {}

  private async decoratePoll(row: Poll) {
    const user = await this.userRepo.findOne({ where: { id: row.user_id } })
    return {
      id: row.id, title: row.title, description: row.description || '',
      options: JSON.parse(row.options || '[]'), vote_count: row.vote_count || 0,
      user_id: row.user_id,
      user: user ? { id: user.id, username: user.username } : undefined,
      created_at: row.created_at, updated_at: row.updated_at,
    }
  }

  async list(page = 1, pageSize = 20) {
    const [rows, total] = await this.pollRepo.findAndCount({
      where: {},
      order: { created_at: 'DESC' },
      skip: (page - 1) * pageSize, take: pageSize,
    })
    const list = await Promise.all(rows.map((r) => this.decoratePoll(r)))
    const totalPage = pageSize > 0 ? Math.ceil(total / pageSize) : 1
    return { list, total, page, page_size: pageSize, total_page: totalPage }
  }

  async detail(id: number, userId?: number, visitorId?: string) {
    const row = await this.pollRepo.findOne({ where: { id } })
    if (!row) throw new NotFoundException('投票不存在')

    const counts: Record<string, number> = {}
    const voteRows = await this.voteRepo
      .createQueryBuilder('pv')
      .select('pv.option_index', 'option_index')
      .addSelect('COUNT(*)', 'c')
      .where('pv.poll_id = :id', { id })
      .groupBy('pv.option_index')
      .getRawMany()
    for (const r of voteRows) counts[String(r.option_index)] = Number(r.c)
    const totalVotes = Object.values(counts).reduce((a, b) => a + b, 0)

    let myVote: number | null = null
    const voteWhere: any = { poll_id: id }
    if (userId) voteWhere.user_id = userId
    else if (visitorId) voteWhere.visitor_id = visitorId
    const existingVote = await this.voteRepo.findOne({ where: voteWhere })
    if (existingVote) myVote = existingVote.option_index

    return { poll: await this.decoratePoll(row), vote_counts: counts, total_votes: totalVotes, my_vote: myVote }
  }

  async vote(id: number, optionIndex: number, userId?: number, visitorId?: string) {
    const row = await this.pollRepo.findOne({ where: { id } })
    if (!row) throw new NotFoundException('投票不存在')
    const options = JSON.parse(row.options || '[]')
    if (!Number.isInteger(optionIndex) || optionIndex < 0 || optionIndex >= options.length)
      throw new BadRequestException('选项不存在')

    const voteWhere: any = { poll_id: id }
    if (userId) voteWhere.user_id = userId
    else if (visitorId) voteWhere.visitor_id = visitorId
    const existing = await this.voteRepo.findOne({ where: voteWhere })

    if (existing) {
      if (existing.option_index !== optionIndex)
        await this.voteRepo.update(existing.id, { option_index: optionIndex })
    } else {
      const v = this.voteRepo.create({ poll_id: id, user_id: userId ?? undefined, visitor_id: visitorId, option_index: optionIndex })
      await this.voteRepo.save(v)
      await this.pollRepo.increment({ id }, 'vote_count', 1)
    }

    return this.detail(id, userId, visitorId)
  }

  async create(title: string, description: string, options: string[], userId: number) {
    if (options.length < 2) throw new BadRequestException('至少需要两个选项')
    const row = this.pollRepo.create({ title: title.trim(), description, options: JSON.stringify(options), user_id: userId, vote_count: 0 })
    const saved = await this.pollRepo.save(row)
    return this.decoratePoll(saved)
  }

  async softDelete(id: number, userId: number, isAdmin: boolean) {
    const row = await this.pollRepo.findOne({ where: { id } })
    if (!row) throw new NotFoundException('投票不存在')
    if (userId !== row.user_id && !isAdmin) throw new ForbiddenException('无权删除该投票')
    await this.pollRepo.softDelete(id)
  }
}
