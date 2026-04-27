/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  defaultAboutPageConfig,
  translateAboutPageConfig,
} from '../pages/About/aboutPageConfig.js';

const localeDir = join(dirname(fileURLToPath(import.meta.url)), 'locales');

const requiredModelMonitorCopyKeys = [
  '点击复制渠道名称',
  '已复制渠道名称',
  '该模型已命中排除规则，无法单独开启定时测试',
  '通知人管理',
  '接收通知',
  '接收',
  '不可接收',
  '未配置邮箱',
  '无模型监控查看权限',
  '取消勾选后，模型失败邮件不会发送给对应管理员。',
];

const requiredAboutPageCopyKeys = [
  '关于页面配置',
  '关于页面配置已更新',
  '关于页面配置更新失败',
  '关于兼容内容已更新',
  '关于兼容内容更新失败',
  '旧版关于内容',
  '填写旧版关于页面 Markdown 或 HTML 内容；结构化配置启用后仍可保留兼容内容',
  '旧版关于内容已更新',
  '旧版关于内容更新失败',
  '关于页面配置/开关',
  '结构化配置用于新版关于页面；高级兼容内容可继续保存旧版 About 配置。',
  '启用结构化关于页面',
  '开',
  '关',
  '首屏内容',
  '眉标',
  '主标题',
  '副标题',
  '主按钮文案',
  '主按钮链接',
  '次按钮文案',
  '次按钮链接',
  '平台概览',
  '概览标题',
  '运行状态',
  '概览描述',
  '指标',
  '指标数值',
  '指标标签',
  '渠道',
  '渠道名称',
  '渠道占比',
  '渠道状态',
  '能力卡片',
  '图标标识',
  '卡片标题',
  '卡片描述',
  '集团背书',
  '集团标题',
  '集团状态',
  '集团描述',
  '背书要点',
  '官网按钮文案',
  '官网链接',
  '客服二维码',
  '微信客服',
  '企业微信客服',
  '客服标题',
  '二维码图片链接',
  '二维码图片地址',
  '备用联系链接',
  '备用链接',
  '客服说明',
  '高级兼容内容',
  '自定义内容',
  '保存关于页面配置',
  '旧版 About 内容仅用于兼容旧页面或回退场景，保存按钮会单独写入 About 配置项。',
  '保存旧版关于内容',
  '二维码暂未配置',
  '联系二维码',
  '企业微信',
  '页面操作',
  '业务概览',
  '核心能力',
  '访问网站',
  '联系渠道',
  '联系方式',
  '神州数码集团 · 企业级 AI 能力入口',
  '统一接入、分发与治理企业 AI 能力',
  '面向企业团队与开发者的一站式 AI API 聚合平台，统一模型接入、鉴权、计费与观测能力。',
  '进入控制台',
  '访问集团官网',
  'AI Gateway Overview',
  '统一协议、统一鉴权、统一计费，降低多模型接入复杂度。',
  '运行中',
  '上游模型渠道',
  '服务可用性目标',
  '企业支持响应',
  '健康',
  '稳定',
  '可用',
  '多模型聚合',
  '统一接入主流大模型服务，减少多供应商集成和维护成本。',
  '智能路由分发',
  '按模型、渠道、分组和策略灵活调度请求，提升调用稳定性。',
  '企业安全治理',
  '集中管理鉴权、额度、分组和审计能力，支撑企业级管控。',
  '用量计费分析',
  '沉淀调用、消耗和账单数据，帮助团队持续优化 AI 成本。',
  '神州数码集团企业数字化能力支撑',
  '依托神州数码在企业数字化服务、云计算和生态整合领域的能力，提供稳定可信的 AI API 聚合入口。',
  '集团能力支持',
  '服务企业数字化转型与智能化升级',
  '整合多云、多模型和多场景 AI 能力',
  '面向业务团队提供统一、可治理的接入体验',
  'Digital China',
  '扫码添加平台客服，获取接入咨询与使用支持。',
  '通过企业微信联系服务团队，获取企业级支持。',
];

const defaultStructuredAboutCopyKeys = [
  ...new Set(
    (() => {
      const keys = [];

      translateAboutPageConfig(defaultAboutPageConfig, (key) => {
        keys.push(key);
        return key;
      });

      return keys;
    })(),
  ),
];

test('all locales include model monitor channel copy keys', () => {
  const localeFiles = readdirSync(localeDir).filter((file) =>
    file.endsWith('.json'),
  );

  assert.ok(localeFiles.length > 0);
  for (const localeFile of localeFiles) {
    const localeFileContent = JSON.parse(
      readFileSync(join(localeDir, localeFile), 'utf8'),
    );
    const locale = localeFileContent.translation || localeFileContent;
    for (const key of requiredModelMonitorCopyKeys) {
      assert.equal(
        Object.prototype.hasOwnProperty.call(locale, key),
        true,
        `${localeFile} missing ${key}`,
      );
    }
  }
});

test('all locales include structured about page copy keys', () => {
  const localeFiles = readdirSync(localeDir).filter((file) =>
    file.endsWith('.json'),
  );

  assert.ok(localeFiles.length > 0);
  for (const localeFile of localeFiles) {
    const localeFileContent = JSON.parse(
      readFileSync(join(localeDir, localeFile), 'utf8'),
    );
    const locale = localeFileContent.translation || localeFileContent;
    for (const key of [
      ...requiredAboutPageCopyKeys,
      ...defaultStructuredAboutCopyKeys,
    ]) {
      assert.equal(
        Object.prototype.hasOwnProperty.call(locale, key),
        true,
        `${localeFile} missing ${key}`,
      );
    }
  }
});
