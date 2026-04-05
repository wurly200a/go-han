# API仕様

## 共通仕様

- ベースURL: `/api`
- リクエスト／レスポンス形式: JSON
- エラー時は `{"error": "<message>"}` を返す

## エンドポイント一覧

| メソッド | パス | 概要 |
|--------|------|------|
| GET | `/api/health` | ヘルスチェック |
| GET | `/api/users` | 全ユーザー一覧（ロール情報含む）取得 |
| PUT | `/api/users/:user_id/roles` | ユーザーのロール更新 |
| GET | `/api/meals` | 指定期間の食事予定一覧取得 |
| PUT | `/api/meals/bulk-update` | 複数食事予定の一括更新 |
| GET | `/api/user-defaults/:user_id` | ユーザーのデフォルト設定取得 |
| PUT | `/api/user-defaults/:user_id` | ユーザーのデフォルト設定更新 |
| GET | `/api/cook-schedules` | 指定期間の料理担当（解決済み）取得 |
| PUT | `/api/cook-schedules` | 日付別料理担当の個別設定 |
| DELETE | `/api/cook-schedules` | 日付別個別設定の削除（デフォルトに戻す） |
| GET | `/api/cook-default-schedules` | 曜日別デフォルト料理担当取得 |
| PUT | `/api/cook-default-schedules` | 曜日別デフォルト料理担当更新 |

## 各エンドポイント詳細

### GET `/api/health`

DBへのping結果を返す。起動確認・死活監視用。

---

### GET `/api/users`

全ユーザーをロール情報付きで返す。

**レスポンス例**

```json
[
  { "id": 1, "name": "Mother", "is_cook": true,  "is_eater": false },
  { "id": 2, "name": "Father", "is_cook": true,  "is_eater": true  },
  { "id": 3, "name": "Taro",   "is_cook": false, "is_eater": true  }
]
```

---

### PUT `/api/users/:user_id/roles`

指定ユーザーの `is_cook` / `is_eater` を更新する。

**リクエストボディ例**

```json
{ "is_cook": true, "is_eater": false }
```

**設計上のポイント**

- 両方 `true` も有効（例: Father）。
- `is_eater=false` にすると `GET /api/meals` の結果から除外される。
- 存在しない `user_id` の場合は `404` を返す。

---

### GET `/api/meals`

指定期間内の全ユーザーの食事予定を返す。

**クエリパラメータ**

| パラメータ | 必須 | 説明 |
|---------|------|------|
| `date` | 必須 | 開始日 (`YYYY-MM-DD`) |
| `days` | 必須 | 取得日数（整数） |

**レスポンス例**

```json
[
  {
    "date": "2024-02-04",
    "user_id": 1,
    "user_name": "Taro",
    "lunch": 2,
    "dinner": 2
  }
]
```

**設計上のポイント**

- `meals` テーブルに登録がない日は、`user_defaults`（曜日別デフォルト）の値を返す。予定がない日でも毎週同じデフォルトを手入力しなくて済むための仕組み。
- 単一SQLクエリでウィンドウ関数を使い、昼・夕の2行を1行にピボットしている。N+1を避けるための設計。

---

### PUT `/api/meals/bulk-update`

複数の食事予定をまとめて登録・更新する。

**リクエストボディ例**

```json
[
  {
    "user_id": 1,
    "date": "2024-02-04",
    "lunch": 2,
    "dinner": 3
  }
]
```

**設計上のポイント**

- 1件の変更もこのエンドポイントに統一（フロントエンドは1件でも配列で送る）。
- トランザクションで一括処理し、途中失敗時はロールバック。
- 変更が **24時間以内の食事** に対するものであれば Slack に通知する。直前変更は家族への影響が大きいため。

**meal_option の値**

| 値 | 意味 |
|----|------|
| 1 | なし（食べない） |
| 2 | 家（自宅で食べる） |
| 3 | 弁当（弁当持参） |

---

### GET `/api/user-defaults/:user_id`

ユーザーの曜日別デフォルト設定（昼・夕）を取得する。

**レスポンス例**

```json
[
  { "day_of_week": 0, "lunch": 2, "dinner": 2 },
  { "day_of_week": 1, "lunch": 2, "dinner": 2 }
]
```

`day_of_week` は 0=日曜〜6=土曜。

---

### PUT `/api/user-defaults/:user_id`

ユーザーの曜日別デフォルト設定を更新する。リクエスト形式はGETのレスポンスと同じ。

---

### GET `/api/cook-schedules`

指定期間内の料理担当を解決済みで返す。優先度：個別設定 → 曜日デフォルト → 各自（null）。

**クエリパラメータ**

| パラメータ | 必須 | 説明 |
|---------|------|------|
| `date` | 必須 | 開始日 (`YYYY-MM-DD`) |
| `days` | 必須 | 取得日数（整数） |

**レスポンス例**

```json
{
  "2026-04-06": {
    "lunch":  { "cook_user_id": 5, "cook_user_name": "Mother" },
    "dinner": null
  },
  "2026-04-07": {
    "lunch":  null,
    "dinner": { "cook_user_id": 2, "cook_user_name": "Father" }
  }
}
```

`null` = 各自。フロントエンドはこの値を見て eater の dropdown / 「各自」テキスト表示を切り替える。

---

### PUT `/api/cook-schedules`

日付別の料理担当を個別設定する（upsert）。`cook_user_id=null` で「各自」を曜日デフォルトより優先して上書き。

```json
[
  { "date": "2026-04-06", "meal_period": 1, "cook_user_id": 5 },
  { "date": "2026-04-06", "meal_period": 2, "cook_user_id": null }
]
```

---

### DELETE `/api/cook-schedules`

個別設定を削除してデフォルトに戻す。

```json
[
  { "date": "2026-04-06", "meal_period": 1 }
]
```

---

### GET `/api/cook-default-schedules`

曜日別デフォルト設定を返す。登録のない曜日×区分は暗黙的に各自。

```json
[
  { "day_of_week": 1, "meal_period": 1, "cook_user_id": 5, "cook_user_name": "Mother" },
  { "day_of_week": 1, "meal_period": 2, "cook_user_id": null, "cook_user_name": null }
]
```

`day_of_week` は 0=日曜〜6=土曜。`meal_period` は 1=昼 / 2=夜。

---

### PUT `/api/cook-default-schedules`

曜日別デフォルト設定を更新する（upsert）。リクエスト形式は `cook_user_name` を除いた GET レスポンスと同じ。
