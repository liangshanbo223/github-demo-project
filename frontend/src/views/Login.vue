<template>
    <v-container class="fill-height" style="margin-top: 100px;">
      <v-row justify="center" align="center">
        <v-col cols="12" sm="8" md="4">
          <v-card class="rounded-xl" elevation="4">
            <v-card-title class="headline">
              {{ isRecoverMode ? '找回/重置账户密码' : $t('login.title') }}
            </v-card-title>
            <v-card-text>
              <v-form v-if="!isRecoverMode" @submit.prevent="login" ref="form">
                <v-text-field v-model="username" :label="$t('login.username')" :rules="usernameRules" required></v-text-field>
                <v-text-field v-model="password" :label="$t('login.password')" :rules="passwordRules" type="password" required></v-text-field>
                <v-btn :loading="loading" type="submit" color="primary" block class="mt-2" v-text="$t('actions.submit')"></v-btn>
                <v-btn variant="text" block class="mt-2 text-none" color="secondary" @click="isRecoverMode = true">
                  忘记密码？使用 Recovery Key 重置
                </v-btn>
              </v-form>

              <v-form v-else @submit.prevent="recover" ref="recoverForm">
                <v-text-field v-model="recoverKey" label="Recovery Key (密钥)" required></v-text-field>
                <v-text-field v-model="newUsername" label="新管理员用户名" required></v-text-field>
                <v-text-field v-model="newPassword" label="新管理员密码" type="password" required></v-text-field>
                <v-btn :loading="loading" type="submit" color="warning" block class="mt-2">
                  重置管理员账户
                </v-btn>
                <v-btn variant="text" block class="mt-2 text-none" @click="isRecoverMode = false">
                  返回登录
                </v-btn>
              </v-form>
              <v-select
                density="compact"
                class="mt-2"
                hide-details
                variant="solo"
                :items="languages"
                v-model="$i18n.locale"
                @update:modelValue="changeLocale">
                <template v-slot:append>
                  <v-menu>
                    <template v-slot:activator="{ props }">
                      <v-btn icon v-bind="props">
                        <v-icon>mdi-theme-light-dark</v-icon>
                      </v-btn>
                    </template>
                    <v-list>
                      <v-list-item
                        v-for="th in themes"
                        :key="th.value"
                        @click="changeTheme(th.value)"
                        :prepend-icon="th.icon"
                        :active="isActiveTheme(th.value)"
                      >
                        <v-list-item-title>{{ $t(`theme.${th.value}`) }}</v-list-item-title>
                      </v-list-item>
                    </v-list>
                  </v-menu>
                </template>
              </v-select>
            </v-card-text>
          </v-card>
        </v-col>
      </v-row>
    </v-container>
  </template>
  
<script lang="ts" setup>
import { ref } from "vue"
import { useLocale,useTheme } from 'vuetify'
import { i18n, languages } from '@/locales'
import { useRouter } from 'vue-router'
import HttpUtil from '@/plugins/httputil'
import { push } from 'notivue'

const theme = useTheme()
const locale = useLocale()

const themes = [
  { value: 'light', icon: 'mdi-white-balance-sunny' },
  { value: 'dark', icon: 'mdi-moon-waning-crescent' },
  { value: 'system', icon: 'mdi-laptop' },
]

const username = ref('')
const usernameRules = [
  (value: string) => {
    if (value?.length > 0) return true
    return i18n.global.t('login.unRules')
  },
]

const password = ref('')
const passwordRules = [
  (value: string) => {
    if (value?.length > 0) return true
    return i18n.global.t('login.pwRules')
  },
]

const isRecoverMode = ref(false)
const recoverKey = ref('')
const newUsername = ref('')
const newPassword = ref('')

const loading = ref(false)
const router = useRouter()

const login = async () => {
  if (username.value == '' || password.value == '') return
  loading.value=true
  const response = await HttpUtil.post('api/login',{user: username.value, pass: password.value})
  if(response.success){
    setTimeout(() => {
      loading.value=false
      router.push('/')
    }, 500)
  } else {
    loading.value=false
  }
}

const recover = async () => {
  if (recoverKey.value == '' || newUsername.value == '' || newPassword.value == '') return
  loading.value = true
  
  const formData = new FormData()
  formData.append('key', recoverKey.value)
  formData.append('username', newUsername.value)
  formData.append('password', newPassword.value)
  
  const response = await HttpUtil.post('api/recovery', formData)
  loading.value = false
  if (response.success) {
    push.success({ message: '管理员账户重置成功，请使用新凭据登录' })
    isRecoverMode.value = false
    username.value = newUsername.value
    password.value = ''
    recoverKey.value = ''
    newUsername.value = ''
    newPassword.value = ''
  }
}

const changeLocale = (l: any) => {
  locale.current.value = l ?? 'zhHans'
  localStorage.setItem('locale', locale.current.value)
}
const changeTheme = (th: string) => {
  theme.change(th)
  localStorage.setItem('theme', th)
}
const isActiveTheme = (th: string) => {
  const current = localStorage.getItem('theme') ?? 'system'
  return current == th
}
</script>
