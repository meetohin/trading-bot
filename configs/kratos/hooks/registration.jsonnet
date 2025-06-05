function(ctx) {
  identity_id: ctx.identity.id,
  email: ctx.identity.traits.email,
  username: ctx.identity.traits.username,
  first_name: if std.objectHas(ctx.identity.traits, 'first_name') then ctx.identity.traits.first_name else '',
  last_name: if std.objectHas(ctx.identity.traits, 'last_name') then ctx.identity.traits.last_name else '',
  subscription_plan: if std.objectHas(ctx.identity.traits, 'subscription_plan') then ctx.identity.traits.subscription_plan else 'free'
}
