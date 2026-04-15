export const CHAT_COMPLETIONS_TO_RESPONSES_POLICY_TEMPLATE = JSON.stringify(
  {
    enabled: true,
    channel_types: [45],
    model_patterns: [
      '^doubao-seed-translation-.*$',
      '^doubao-seed-1-6-thinking-.*$',
    ],
  },
  null,
  2,
);

export const CHAT_COMPLETIONS_TO_RESPONSES_POLICY_ALL_CHANNELS_EXAMPLE =
  JSON.stringify(
    {
      enabled: true,
      all_channels: true,
      model_patterns: [
        '^doubao-seed-translation-.*$',
        '^doubao-seed-1-6-thinking-.*$',
      ],
    },
    null,
    2,
  );

export const DEFAULT_GLOBAL_SETTING_INPUTS = {
  'global.pass_through_request_enabled': false,
  'global.thinking_model_blacklist': '[]',
  'global.chat_completions_to_responses_policy':
    CHAT_COMPLETIONS_TO_RESPONSES_POLICY_TEMPLATE,
  'general_setting.ping_interval_enabled': false,
  'general_setting.ping_interval_seconds': 60,
};
