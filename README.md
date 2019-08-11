# TheMAP Test
### Карты
| Номер  | Месяц  |  Год | CVV  | Владелец  | Баланс  |
| :----------: | :----------: | :----------: | :----------: | :----------: | :----------: |
| 4012888888881881  | 9  | 2019  | 100  | Ivanon Ivan  | 200  |
| 4539068477119696  | 9 | 2019 | 100  | Ivanon Ivan  | 200  |
| 5192679737272623  | 9 | 2019 | 100  | Ivanon Ivan  | 0  |
| 5420893164982661  | 9 | 2019 | 100  | Ivanon Ivan  | 0  |
| 344681483420255  | 9 | 2019 | 100  | Ivanon Ivan  | 1000  |
| 341487100962668  | 9 | 2019 | 100  | Ivanon Ivan  | 1000  |
| 6011755772471507  | 9 | 2019 | 100  | Ivanon Ivan  | 68000  |
| 6011937144761860  | 9 | 2019 | 100  | Ivanon Ivan  | 68315  |

####Пример запроса к /block
Method = "POST"
#####Headers
Content-Type = "application/json"
#####JSON
```JSON
{
   "merchant_contract_id":128,
   "card":{
      "pan":"4012888888881881",
      "e_month":9,
      "e_year":2019,
      "cvv":100,
      "holder":"IVANOV IVAN"
   },
   "deal":{
      "order_id":"48eg6",
      "amount":1
   }
}```
####Пример запроса к /charge
Method = "POST"
#####Headers
Content-Type = "application/json"
#####JSON
```JSON
{
   "deal_id":1,
   "amount":100
}
```
