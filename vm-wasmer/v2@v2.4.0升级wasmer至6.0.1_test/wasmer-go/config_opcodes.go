package wasmer

// #include <wasmer.h>
import "C"

//type Opcode C.wasmer_parser_operator_t
//
//const (
//	Unreachable        Opcode = 0
//	Nop                       = 1
//	Block                     = 2
//	Loop                      = 3
//	If                        = 4
//	Else                      = 5
//	Try                       = 6
//	Catch                     = 7
//	CatchAll                  = 8
//	Delegate                  = 9
//	Throw                     = 10
//	Rethrow                   = 11
//	Unwind                    = 12
//	End                       = 13
//	Br                        = 14
//	BrIf                      = 15
//	BrTable                   = 16
//	Return                    = 17
//	Call                      = 18
//	CallIndirect              = 19
//	ReturnCall                = 20
//	ReturnCallIndirect        = 21
//	Drop                      = 22
//	Select                    = 23
//	TypedSelect               = 24
//	LocalGet                  = 25
//	LocalSet                  = 26
//	LocalTee                  = 27
//	GlobalGet                 = 28
//	GlobalSet                 = 29
//	I32Load                   = 30
//	I64Load                   = 31
//	F32Load                   = 32
//	F64Load                   = 33
//	I32Load8S                 = 34
//	I32Load8U                 = 35
//	I32Load16S                = 36
//	I32Load16U                = 37
//	I64Load8S                 = 38
//	I64Load8U                 = 39
//	I64Load16S                = 40
//	I64Load16U                = 41
//	I64Load32S                = 42
//	I64Load32U                = 43
//	I32Store                  = 44
//	I64Store                  = 45
//	F32Store                  = 46
//	F64Store                  = 47
//	I32Store8                 = 48
//	I32Store16                = 49
//	I64Store8                 = 50
//	I64Store16                = 51
//	I64Store32                = 52
//	MemorySize                = 53
//	MemoryGrow                = 54
//	I32Const                  = 55
//	I64Const                  = 56
//	F32Const                  = 57
//	F64Const                  = 58
//	RefNull                   = 59
//	RefIsNull                 = 60
//	RefFunc                   = 61
//	I32Eqz                    = 62
//	I32Eq                     = 63
//	I32Ne                     = 64
//	I32LtS                    = 65
//	I32LtU                    = 66
//	I32GtS                    = 67
//	I32GtU                    = 68
//	I32LeS                    = 69
//	I32LeU                    = 70
//	I32GeS                    = 71
//	I32GeU                    = 72
//	I64Eqz                    = 73
//	I64Eq                     = 74
//	I64Ne                     = 75
//	I64LtS                    = 76
//	I64LtU                    = 77
//	I64GtS                    = 78
//	I64GtU                    = 79
//	I64LeS                    = 80
//	I64LeU                    = 81
//	I64GeS                    = 82
//	I64GeU                    = 83
//	F32Eq                     = 84
//	F32Ne                     = 85
//	F32Lt                     = 86
//	F32Gt                     = 87
//	F32Le                     = 88
//	F32Ge                     = 89
//	F64Eq                     = 90
//	F64Ne                     = 91
//	F64Lt                     = 92
//	F64Gt                     = 93
//	F64Le                     = 94
//	F64Ge                     = 95
//	I32Clz                    = 96
//	I32Ctz                    = 97
//	I32Popcnt                 = 98
//	I32Add                    = 99
//	I32Sub                    = 100
//	I32Mul                    = 101
//	I32DivS                   = 102
//	I32DivU                   = 103
//	I32RemS                   = 104
//	I32RemU                   = 105
//	I32And                    = 106
//	I32Or                     = 107
//	I32Xor                    = 108
//	I32Shl                    = 109
//	I32ShrS                   = 110
//	I32ShrU                   = 111
//	I32Rotl                   = 112
//	I32Rotr                   = 113
//	I64Clz                    = 114
//	I64Ctz                    = 115
//	I64Popcnt                 = 116
//	I64Add                    = 117
//	I64Sub                    = 118
//	I64Mul                    = 119
//	I64DivS                   = 120
//	I64DivU                   = 121
//	I64RemS                   = 122
//	I64RemU                   = 123
//	I64And                    = 124
//	I64Or                     = 125
//	I64Xor                    = 126
//	I64Shl                    = 127
//	I64ShrS                   = 128
//	I64ShrU                   = 129
//	I64Rotl                   = 130
//	I64Rotr                   = 131
//	F32Abs                    = 132
//	F32Neg                    = 133
//	F32Ceil                   = 134
//	F32Floor                  = 135
//	F32Trunc                  = 136
//	F32Nearest                = 137
//	F32Sqrt                   = 138
//	F32Add                    = 139
//	F32Sub                    = 140
//	F32Mul                    = 141
//	F32Div                    = 142
//	F32Min                    = 143
//	F32Max                    = 144
//	F32Copysign               = 145
//	F64Abs                    = 146
//	F64Neg                    = 147
//	F64Ceil                   = 148
//	F64Floor                  = 149
//	F64Trunc                  = 150
//	F64Nearest                = 151
//	F64Sqrt                   = 152
//	F64Add                    = 153
//	F64Sub                    = 154
//	F64Mul                    = 155
//	F64Div                    = 156
//	F64Min                    = 157
//	F64Max                    = 158
//	F64Copysign               = 159
//	I32WrapI64                = 160
//	I32TruncF32S              = 161
//	I32TruncF32U              = 162
//	I32TruncF64S              = 163
//	I32TruncF64U              = 164
//	I64ExtendI32S             = 165
//	I64ExtendI32U             = 166
//	I64TruncF32S              = 167
//	I64TruncF32U              = 168
//	I64TruncF64S              = 169
//	I64TruncF64U              = 170
//	F32ConvertI32S            = 171
//	F32ConvertI32U            = 172
//	F32ConvertI64S            = 173
//	F32ConvertI64U            = 174
//	F32DemoteF64              = 175
//	F64ConvertI32S            = 176
//	F64ConvertI32U            = 177
//	F64ConvertI64S            = 178
//	F64ConvertI64U            = 179
//	F64PromoteF32             = 180
//	I32ReinterpretF32         = 181
//	I64ReinterpretF64         = 182
//	F32ReinterpretI32         = 183
//	F64ReinterpretI64         = 184
//	I32Extend8S               = 185
//	I32Extend16S              = 186
//	I64Extend8S               = 187
//	I64Extend16S              = 188
//	I64Extend32S              = 189
//	I32TruncSatF32S           = 190
//	I32TruncSatF32U           = 191
//	I32TruncSatF64S           = 192
//	I32TruncSatF64U           = 193
//	I64TruncSatF32S           = 194
//	I64TruncSatF32U           = 195
//	I64TruncSatF64S           = 196
//	I64TruncSatF64U           = 197
//	MemoryInit                = 198
//	DataDrop                  = 199
//	MemoryCopy                = 200
//	MemoryFill                = 201
//	TableInit                 = 202
//	ElemDrop                  = 203
//	TableCopy                 = 204
//	TableFill                 = 205
//	TableGet                  = 206
//	TableSet                  = 207
//	TableGrow                 = 208
//	// REVIEW
//	OpTableSize               = 209
//	MemoryAtomicNotify        = 210
//	MemoryAtomicWait32        = 211
//	MemoryAtomicWait64        = 212
//	AtomicFence               = 213
//	I32AtomicLoad             = 214
//	I64AtomicLoad             = 215
//	I32AtomicLoad8U           = 216
//	I32AtomicLoad16U          = 217
//	I64AtomicLoad8U           = 218
//	I64AtomicLoad16U          = 219
//	I64AtomicLoad32U          = 220
//	I32AtomicStore            = 221
//	I64AtomicStore            = 222
//	I32AtomicStore8           = 223
//	I32AtomicStore16          = 224
//	I64AtomicStore8           = 225
//	I64AtomicStore16          = 226
//	I64AtomicStore32          = 227
//	I32AtomicRmwAdd           = 228
//	I64AtomicRmwAdd           = 229
//	I32AtomicRmw8AddU         = 230
//	I32AtomicRmw16AddU        = 231
//	I64AtomicRmw8AddU         = 232
//	I64AtomicRmw16AddU        = 233
//	I64AtomicRmw32AddU        = 234
//	I32AtomicRmwSub           = 235
//	I64AtomicRmwSub           = 236
//	I32AtomicRmw8SubU         = 237
//	I32AtomicRmw16SubU        = 238
//	I64AtomicRmw8SubU         = 239
//	I64AtomicRmw16SubU        = 240
//	I64AtomicRmw32SubU        = 241
//	I32AtomicRmwAnd           = 242
//	I64AtomicRmwAnd           = 243
//	I32AtomicRmw8AndU         = 244
//	I32AtomicRmw16AndU        = 245
//	I64AtomicRmw8AndU         = 246
//	I64AtomicRmw16AndU        = 247
//	I64AtomicRmw32AndU        = 248
//	I32AtomicRmwOr            = 249
//	I64AtomicRmwOr            = 250
//	I32AtomicRmw8OrU          = 251
//	I32AtomicRmw16OrU         = 252
//	I64AtomicRmw8OrU          = 253
//	I64AtomicRmw16OrU         = 254
//	I64AtomicRmw32OrU         = 255
//	I32AtomicRmwXor           = 256
//	I64AtomicRmwXor           = 257
//	I32AtomicRmw8XorU         = 258
//	I32AtomicRmw16XorU        = 259
//	I64AtomicRmw8XorU         = 260
//	I64AtomicRmw16XorU        = 261
//	I64AtomicRmw32XorU        = 262
//	I32AtomicRmwXchg          = 263
//	I64AtomicRmwXchg          = 264
//	I32AtomicRmw8XchgU        = 265
//	I32AtomicRmw16XchgU       = 266
//	I64AtomicRmw8XchgU        = 267
//	I64AtomicRmw16XchgU       = 268
//	I64AtomicRmw32XchgU       = 269
//	I32AtomicRmwCmpxchg       = 270
//	I64AtomicRmwCmpxchg       = 271
//	I32AtomicRmw8CmpxchgU     = 272
//	I32AtomicRmw16CmpxchgU    = 273
//	I64AtomicRmw8CmpxchgU     = 274
//	I64AtomicRmw16CmpxchgU    = 275
//	I64AtomicRmw32CmpxchgU    = 276
//	V128Load                  = 277
//	V128Store                 = 278
//	V128Const                 = 279
//	I8x16Splat                = 280
//	I8x16ExtractLaneS         = 281
//	I8x16ExtractLaneU         = 282
//	I8x16ReplaceLane          = 283
//	I16x8Splat                = 284
//	I16x8ExtractLaneS         = 285
//	I16x8ExtractLaneU         = 286
//	I16x8ReplaceLane          = 287
//	I32x4Splat                = 288
//	I32x4ExtractLane          = 289
//	I32x4ReplaceLane          = 290
//	I64x2Splat                = 291
//	I64x2ExtractLane          = 292
//	I64x2ReplaceLane          = 293
//	F32x4Splat                = 294
//	F32x4ExtractLane          = 295
//	F32x4ReplaceLane          = 296
//	F64x2Splat                = 297
//	F64x2ExtractLane          = 298
//	F64x2ReplaceLane          = 299
//	I8x16Eq                   = 300
//	I8x16Ne                   = 301
//	I8x16LtS                  = 302
//	I8x16LtU                  = 303
//	I8x16GtS                  = 304
//	I8x16GtU                  = 305
//	I8x16LeS                  = 306
//	I8x16LeU                  = 307
//	I8x16GeS                  = 308
//	I8x16GeU                  = 309
//	I16x8Eq                   = 310
//	I16x8Ne                   = 311
//	I16x8LtS                  = 312
//	I16x8LtU                  = 313
//	I16x8GtS                  = 314
//	I16x8GtU                  = 315
//	I16x8LeS                  = 316
//	I16x8LeU                  = 317
//	I16x8GeS                  = 318
//	I16x8GeU                  = 319
//	I32x4Eq                   = 320
//	I32x4Ne                   = 321
//	I32x4LtS                  = 322
//	I32x4LtU                  = 323
//	I32x4GtS                  = 324
//	I32x4GtU                  = 325
//	I32x4LeS                  = 326
//	I32x4LeU                  = 327
//	I32x4GeS                  = 328
//	I32x4GeU                  = 329
//	I64x2Eq                   = 330
//	I64x2Ne                   = 331
//	I64x2LtS                  = 332
//	I64x2GtS                  = 333
//	I64x2LeS                  = 334
//	I64x2GeS                  = 335
//	F32x4Eq                   = 336
//	F32x4Ne                   = 337
//	F32x4Lt                   = 338
//	F32x4Gt                   = 339
//	F32x4Le                   = 340
//	F32x4Ge                   = 341
//	F64x2Eq                   = 342
//	F64x2Ne                   = 343
//	F64x2Lt                   = 344
//	F64x2Gt                   = 345
//	F64x2Le                   = 346
//	F64x2Ge                   = 347
//	V128Not                   = 348
//	V128And                   = 349
//	V128AndNot                = 350
//	V128Or                    = 351
//	V128Xor                   = 352
//	V128Bitselect             = 353
//	V128AnyTrue               = 354
//	I8x16Abs                  = 355
//	I8x16Neg                  = 356
//	I8x16AllTrue              = 357
//	I8x16Bitmask              = 358
//	I8x16Shl                  = 359
//	I8x16ShrS                 = 360
//	I8x16ShrU                 = 361
//	I8x16Add                  = 362
//	I8x16AddSatS              = 363
//	I8x16AddSatU              = 364
//	I8x16Sub                  = 365
//	I8x16SubSatS              = 366
//	I8x16SubSatU              = 367
//	I8x16MinS                 = 368
//	I8x16MinU                 = 369
//	I8x16MaxS                 = 370
//	I8x16MaxU                 = 371
//	I8x16Popcnt               = 372
//	I16x8Abs                  = 373
//	I16x8Neg                  = 374
//	I16x8AllTrue              = 375
//	I16x8Bitmask              = 376
//	I16x8Shl                  = 377
//	I16x8ShrS                 = 378
//	I16x8ShrU                 = 379
//	I16x8Add                  = 380
//	I16x8AddSatS              = 381
//	I16x8AddSatU              = 382
//	I16x8Sub                  = 383
//	I16x8SubSatS              = 384
//	I16x8SubSatU              = 385
//	I16x8Mul                  = 386
//	I16x8MinS                 = 387
//	I16x8MinU                 = 388
//	I16x8MaxS                 = 389
//	I16x8MaxU                 = 390
//	I16x8ExtAddPairwiseI8x16S = 391
//	I16x8ExtAddPairwiseI8x16U = 392
//	I32x4Abs                  = 393
//	I32x4Neg                  = 394
//	I32x4AllTrue              = 395
//	I32x4Bitmask              = 396
//	I32x4Shl                  = 397
//	I32x4ShrS                 = 398
//	I32x4ShrU                 = 399
//	I32x4Add                  = 400
//	I32x4Sub                  = 401
//	I32x4Mul                  = 402
//	I32x4MinS                 = 403
//	I32x4MinU                 = 404
//	I32x4MaxS                 = 405
//	I32x4MaxU                 = 406
//	I32x4DotI16x8S            = 407
//	I32x4ExtAddPairwiseI16x8S = 408
//	I32x4ExtAddPairwiseI16x8U = 409
//	I64x2Abs                  = 410
//	I64x2Neg                  = 411
//	I64x2AllTrue              = 412
//	I64x2Bitmask              = 413
//	I64x2Shl                  = 414
//	I64x2ShrS                 = 415
//	I64x2ShrU                 = 416
//	I64x2Add                  = 417
//	I64x2Sub                  = 418
//	I64x2Mul                  = 419
//	F32x4Ceil                 = 420
//	F32x4Floor                = 421
//	F32x4Trunc                = 422
//	F32x4Nearest              = 423
//	F64x2Ceil                 = 424
//	F64x2Floor                = 425
//	F64x2Trunc                = 426
//	F64x2Nearest              = 427
//	F32x4Abs                  = 428
//	F32x4Neg                  = 429
//	F32x4Sqrt                 = 430
//	F32x4Add                  = 431
//	F32x4Sub                  = 432
//	F32x4Mul                  = 433
//	F32x4Div                  = 434
//	F32x4Min                  = 435
//	F32x4Max                  = 436
//	F32x4PMin                 = 437
//	F32x4PMax                 = 438
//	F64x2Abs                  = 439
//	F64x2Neg                  = 440
//	F64x2Sqrt                 = 441
//	F64x2Add                  = 442
//	F64x2Sub                  = 443
//	F64x2Mul                  = 444
//	F64x2Div                  = 445
//	F64x2Min                  = 446
//	F64x2Max                  = 447
//	F64x2PMin                 = 448
//	F64x2PMax                 = 449
//	I32x4TruncSatF32x4S       = 450
//	I32x4TruncSatF32x4U       = 451
//	F32x4ConvertI32x4S        = 452
//	F32x4ConvertI32x4U        = 453
//	I8x16Swizzle              = 454
//	I8x16Shuffle              = 455
//	V128Load8Splat            = 456
//	V128Load16Splat           = 457
//	V128Load32Splat           = 458
//	V128Load32Zero            = 459
//	V128Load64Splat           = 460
//	V128Load64Zero            = 461
//	I8x16NarrowI16x8S         = 462
//	I8x16NarrowI16x8U         = 463
//	I16x8NarrowI32x4S         = 464
//	I16x8NarrowI32x4U         = 465
//	I16x8ExtendLowI8x16S      = 466
//	I16x8ExtendHighI8x16S     = 467
//	I16x8ExtendLowI8x16U      = 468
//	I16x8ExtendHighI8x16U     = 469
//	I32x4ExtendLowI16x8S      = 470
//	I32x4ExtendHighI16x8S     = 471
//	I32x4ExtendLowI16x8U      = 472
//	I32x4ExtendHighI16x8U     = 473
//	I64x2ExtendLowI32x4S      = 474
//	I64x2ExtendHighI32x4S     = 475
//	I64x2ExtendLowI32x4U      = 476
//	I64x2ExtendHighI32x4U     = 477
//	I16x8ExtMulLowI8x16S      = 478
//	I16x8ExtMulHighI8x16S     = 479
//	I16x8ExtMulLowI8x16U      = 480
//	I16x8ExtMulHighI8x16U     = 481
//	I32x4ExtMulLowI16x8S      = 482
//	I32x4ExtMulHighI16x8S     = 483
//	I32x4ExtMulLowI16x8U      = 484
//	I32x4ExtMulHighI16x8U     = 485
//	I64x2ExtMulLowI32x4S      = 486
//	I64x2ExtMulHighI32x4S     = 487
//	I64x2ExtMulLowI32x4U      = 488
//	I64x2ExtMulHighI32x4U     = 489
//	V128Load8x8S              = 490
//	V128Load8x8U              = 491
//	V128Load16x4S             = 492
//	V128Load16x4U             = 493
//	V128Load32x2S             = 494
//	V128Load32x2U             = 495
//	V128Load8Lane             = 496
//	V128Load16Lane            = 497
//	V128Load32Lane            = 498
//	V128Load64Lane            = 499
//	V128Store8Lane            = 500
//	V128Store16Lane           = 501
//	V128Store32Lane           = 502
//	V128Store64Lane           = 503
//	I8x16RoundingAverageU     = 504
//	I16x8RoundingAverageU     = 505
//	I16x8Q15MulrSatS          = 506
//	F32x4DemoteF64x2Zero      = 507
//	F64x2PromoteLowF32x4      = 508
//	F64x2ConvertLowI32x4S     = 509
//	F64x2ConvertLowI32x4U     = 510
//	I32x4TruncSatF64x2SZero   = 511
//	I32x4TruncSatF64x2UZero   = 512
//)
